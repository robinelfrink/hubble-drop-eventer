# Hubble Drop Eventer

Creates [Kubernetes Events](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/)
for dropped packages. Connects to [Hubble](https://github.com/cilium/hubble)
to listen for packet drops.

Although this code works for me, it is in no way production ready.

Example:

```shell
$ kubectl get events --namespace vault
LAST SEEN   TYPE      REASON       OBJECT                                   MESSAGE
48m         Warning   PacketDrop   pod/vault-vaultwarden-6fb76579c5-rxwfv   Incoming packet dropped from ingress-nginx/ingress-nginx-controller-f5f6m (10.244.0.93) port 8080/TCP
```

## Installation

Substitute `<namespace>` with the namespace your `hubble-relay` lives in.

```shell
$ kubectl kustomize ./manifests/ | \
      NAMESPACE=<namespace> envsubst | \
      kubectl apply --filename -
```

## To do

List of things to do, non-exhaustive:

*  Better use of contexts
*  Configuration flags:
   *  History expiration
   *  Namespace include/exclude
*  Verbose/debug output
*  Tests
*  Publish container to ghcr.io
*  Helm chart
