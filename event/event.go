package event

import (
	"fmt"
	"strings"
	"time"

	"github.com/cilium/cilium/api/v1/flow"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
)

type DropEvent struct {
	Pod         string
	Namespace   string
	Direction   flow.TrafficDirection
	Source      map[string]string
	Destination map[string]string
}

type Eventer struct {
	kubeClient *kubernetes.Clientset
	history    map[string]int64
	recorder   record.EventRecorder
	channel    <-chan DropEvent
}

func New(channel <-chan DropEvent) (*Eventer, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeConfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	}
	kubeClient := kubernetes.NewForConfigOrDie(config)
	eventer := Eventer{
		kubeClient: kubeClient,
		history:    map[string]int64{},
		channel:    channel,
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	eventer.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "hubble-drop-eventer"})

	return &eventer, nil

}

func (e *Eventer) ExpireHistory() {
	now := time.Now().Unix()
	for combo, timestamp := range e.history {
		// Poor man's rate limiter. Also remove future entries due to DST.
		if timestamp < now-120 || timestamp > now {
			delete(e.history, combo)
		}
	}
}

func (e *Eventer) Run() {
	for drop := range e.channel {
		e.ExpireHistory()
		if _, exists := e.history[fmt.Sprintf("%s/%s", drop.Source["ip"], drop.Destination["ip"])]; !exists {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      drop.Pod,
					Namespace: drop.Namespace,
				},
			}

			e.recorder.Event(&pod, v1.EventTypeWarning, "PacketDrop", e.createMessage(drop))

			// Add src/dst combo to the history
			e.history[fmt.Sprintf("%s/%s", drop.Source["ip"], drop.Destination["ip"])] = time.Now().Unix()
		}
	}
}

func (e *Eventer) createMessage(drop DropEvent) string {
	var message []string

	if drop.Direction == flow.TrafficDirection_INGRESS {
		message = []string{"Incoming packet dropped from"}
		if drop.Source["pod"] != "" && drop.Source["namespace"] != "" {
			message = append(message, fmt.Sprintf("%s/%s (%s)", drop.Source["namespace"], drop.Source["pod"], drop.Source["ip"]))
		} else if drop.Source["reserved"] != "" {
			message = append(message, fmt.Sprintf("%s (%s)", drop.Source["reserved"], drop.Source["ip"]))
		} else {
			message = append(message, drop.Source["ip"])
		}
	} else {
		message = []string{"Outgoing packet dropped to"}
		if drop.Destination["pod"] != "" && drop.Destination["namespace"] != "" {
			message = append(message, fmt.Sprintf("%s/%s (%s)", drop.Destination["namespace"], drop.Destination["pod"], drop.Destination["ip"]))
		} else if drop.Destination["reserved"] != "" {
			message = append(message, fmt.Sprintf("%s (%s)", drop.Destination["reserved"], drop.Destination["ip"]))
		} else {
			message = append(message, drop.Destination["ip"])
		}
	}

	if drop.Destination["protocol"] != "" {
		if drop.Destination["port"] != "" {
			message = append(message, fmt.Sprintf("port %s/%s", drop.Destination["port"], drop.Destination["protocol"]))
		} else {
			message = append(message, fmt.Sprintf("protocol %s", drop.Destination["protocol"]))
		}
	}

	return strings.Join(message, " ")
}
