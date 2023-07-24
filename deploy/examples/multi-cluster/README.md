1. Create oc aliases for both clusters:
```shell
export KUBECONFIG_WEST=...
```
```shell
export KUBECONFIG_EAST=...
```
```shell
alias oc-west="$KUBECONFIG=<path-to-your-west-cluster-kubeconfig> oc"
```
```shell
alias oc-east="$KUBECONFIG=<path-to-your-east-cluster-kubeconfig> oc"
```

2. Create cacerts secrets:
```shell
oc-west create namespace istio-system
oc-west create secret generic cacerts -n istio-system \
      --from-file=cluster1/ca-cert.pem \
      --from-file=cluster1/ca-key.pem \
      --from-file=cluster1/root-cert.pem \
      --from-file=cluster1/cert-chain.pem
```

```shell
oc-east create namespace istio-system
oc-east create secret generic cacerts -n istio-system \
      --from-file=cluster2/ca-cert.pem \
      --from-file=cluster2/ca-key.pem \
      --from-file=cluster2/root-cert.pem \
      --from-file=cluster2/cert-chain.pem
```

3. Deploy SMCP (TODO: Disable federation controllers):
```shell
sed "s/{{clusterNamePrefix}}/west/g" smcp.tmpl.yaml | oc-west apply -n istio-system -f -
sed "s/{{clusterNamePrefix}}/east/g" smcp.tmpl.yaml | oc-east apply -n istio-system -f -
```

4. Generate kubeconfigs for remote clusters:
```shell
./generate-kubeconfig.sh > istiod-basic-west-cluster.kubeconfig
./generate-kubeconfig.sh > istiod-basic-east-cluster.kubeconfig
```

5. Create secrets from generated kubeconfig:
```shell
oc-west create secret generic istio-remote-secret-east-cluster \
  -n istio-system \
  --from-file=east-cluster=istiod-basic-east-cluster.kubeconfig \
  --type=string
oc-west annotate secret istio-remote-secret-east-cluster -n istio-system networking.istio.io/cluster='east-cluster'
oc-west label secret istio-remote-secret-east-cluster -n istio-system istio/multiCluster='true'
```
```shell
oc-east create secret generic istio-remote-secret-west-cluster \
  -n istio-system \
  --from-file=west-cluster=istiod-basic-west-cluster.kubeconfig \
  --type=string
oc-east annotate secret istio-remote-secret-west-cluster -n istio-system networking.istio.io/cluster='west-cluster'
oc-east label secret istio-remote-secret-west-cluster -n istio-system istio/multiCluster='true'
```

6. Deploy bookinfo on west cluster:
```shell
oc-west new-project bookinfo
oc-west label namespace bookinfo istio-injection=enabled
oc-west apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/platform/kube/bookinfo.yaml -n bookinfo
```

7. Create auto passthrough gateway for east-west gateway:
```shell
oc-west apply -n istio-system -f auto-passthrough-gateway.yaml
oc-east apply -n istio-system -f auto-passthrough-gateway.yaml
```

#### Identified issues:
1. Istio Operator does not create service account `istio-reader-service-account` that should be used by remote cluster.
2. Ingress and egress gateways cannot be disabled when `spec.multi-cluster` is specified.
