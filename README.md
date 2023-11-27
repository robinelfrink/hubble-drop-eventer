# Hubble Drop Eventer

Creates [Kubernetes Events](https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/)
for dropped packages. Connects to [Hubble](https://github.com/cilium/hubble)
to listen for packet drops.

Although this code works for me, it is in no way production ready.

## Installation

Substitute `<namespace>` with the namespace your `hubble-relay` lives in.

```shell
$ kubectl kustomize ~/hubble-drop-eventer/manifests/ | \
      NAMESPACE=<namespace> envsubst | \
      kubectl apply --filename -
```

## To do

*  Better use of contexts
*  Configuration flags:
   *  History expiration
   *  Namespace include/exclude
*  Verbose/debug output
*  Tests
*  Publish container to ghcr.io
*  Helm chart
