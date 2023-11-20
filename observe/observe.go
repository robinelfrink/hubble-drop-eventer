package observe

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/cilium/cilium/api/v1/flow"
	observerpb "github.com/cilium/cilium/api/v1/observer"
	"github.com/cilium/cilium/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	event "hubble-drop-eventer/event"
)

type Observer struct {
	ctx     context.Context
	client  observerpb.ObserverClient
	channel chan<- event.DropEvent
}

func New(server string, port int, channel chan<- event.DropEvent) (*Observer, error) {
	grpcDialOptions := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithReturnConnectionError(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("dns:%s:%d", server, port), grpcDialOptions...)
	if err != nil {
		log.Printf("DialContext")
		return nil, err
	}

	client := observerpb.NewObserverClient(conn)
	log.Println("Connected to Hubble.")

	return &Observer{
		ctx:     ctx,
		client:  client,
		channel: channel,
	}, nil
}

func (o *Observer) Run() {
	request := observerpb.GetFlowsRequest{
		Follow: true,
	}
	stream, err := o.client.GetFlows(o.ctx, &request)
	if err != nil {
		log.Printf("GetFlows")
		return
	}

	for {
		response, err := stream.Recv()
		switch err {
		case io.EOF, context.Canceled:
			log.Printf("%v", err)
			return
		case nil:
		default:
			if status.Code(err) == codes.Canceled {
				log.Printf("%v", err)
				return
			}
			continue
		}

		switch response.GetResponseTypes().(type) {
		case *observerpb.GetFlowsResponse_NodeStatus:
			nodes := strings.Join(response.GetNodeStatus().NodeNames, ", ")
			log.Printf("Observing nodes %s", nodes)
		case *observerpb.GetFlowsResponse_Flow:
			currentFlow := response.GetFlow()
			if currentFlow.Verdict == observerpb.Verdict_DROPPED {
				if err = o.Handle(currentFlow); err != nil {
					log.Printf("%v", err)
					return
				}
			}
		}
	}
}

func (o *Observer) Handle(f *flow.Flow) error {
	var pod, namespace string
	if f.TrafficDirection == observerpb.TrafficDirection_INGRESS {
		pod = f.Destination.PodName
		namespace = f.Destination.Namespace
	} else {
		pod = f.Source.PodName
		namespace = f.Destination.Namespace
	}

	source := ParseEndpoint(f.IP.Source, f.Source, nil)
	destination := ParseEndpoint(f.IP.Destination, f.Destination, f.L4)

	dropEvent := event.DropEvent{
		Pod:         pod,
		Namespace:   namespace,
		Direction:   f.TrafficDirection,
		Source:      source,
		Destination: destination,
	}
	o.channel <- dropEvent

	return nil
}

func ParseEndpoint(ip string, endpoint *flow.Endpoint, l4 *flow.Layer4) map[string]string {
	parts := map[string]string{
		"ip": ip,
	}

	if endpoint.Namespace != "" {
		parts["namespace"] = endpoint.Namespace
	}
	if endpoint.PodName != "" {
		parts["pod"] = endpoint.PodName
	}

	endpointIdentity := identity.NumericIdentity(endpoint.Identity)
	if endpointIdentity.IsReservedIdentity() {
		parts["reserved"] = endpointIdentity.String()
	}

	if l4 != nil {
		switch l4.Protocol.(type) {
		case *flow.Layer4_TCP:
			parts["protocol"] = "TCP"
			parts["port"] = strconv.FormatUint(uint64(l4.GetTCP().DestinationPort), 10)
		case *flow.Layer4_UDP:
			parts["protocol"] = "UDP"
			parts["port"] = strconv.FormatUint(uint64(l4.GetUDP().DestinationPort), 10)
		case *flow.Layer4_ICMPv4:
			parts["protocol"] = "ICMPv4"
		case *flow.Layer4_ICMPv6:
			parts["protocol"] = "ICMPv6"
		}
	}

	return parts
}
