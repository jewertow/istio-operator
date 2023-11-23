1. Export locations of kubeconfig files and create aliases:
```shell
export KUBECONFIG_WEST=
```
```shell
export KUBECONFIG_EAST=
```
```shell
# this is an output directory for kubeconfig files created for istiod to watch remote kube API server
export KUBECONFIG_LOCATION=
```
```shell
alias oc-west="KUBECONFIG=$KUBECONFIG_WEST oc"
alias oc-east="KUBECONFIG=$KUBECONFIG_EAST oc"
```

2. Create cacerts secrets:
```shell
oc-west create namespace istio-system
oc-west label namespace istio-system topology.istio.io/network=network1
oc-west create secret generic cacerts -n istio-system \
      --from-file=cluster1/ca-cert.pem \
      --from-file=cluster1/ca-key.pem \
      --from-file=cluster1/root-cert.pem \
      --from-file=cluster1/cert-chain.pem
oc-east create namespace istio-system
oc-east label namespace istio-system topology.istio.io/network=network2
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

4. Create auto passthrough gateway for east-west gateway:
```shell
sed "s/{{clusterNamePrefix}}/west/g" istio-eastwestgateway.yaml | oc-west apply -n istio-system -f -
sed "s/{{clusterNamePrefix}}/east/g" istio-eastwestgateway.yaml | oc-east apply -n istio-system -f -
```

5. Generate kubeconfigs for remote clusters:
```shell
./generate-kubeconfig.sh \
  --cluster-name=west \
  --namespace=istio-system \
  --revision=basic \
  --remote-kubeconfig-path=$KUBECONFIG_WEST > $KUBECONFIG_LOCATION/istiod-basic-west-cluster.kubeconfig
./generate-kubeconfig.sh \
  --cluster-name=east \
  --namespace=istio-system \
  --revision=basic \
  --remote-kubeconfig-path=$KUBECONFIG_EAST > $KUBECONFIG_LOCATION/istiod-basic-east-cluster.kubeconfig
```

6. Create secrets from generated kubeconfig:
```shell
oc-west create secret generic istio-remote-secret-east-cluster \
  -n istio-system \
  --from-file=east-cluster=$KUBECONFIG_LOCATION/istiod-basic-east-cluster.kubeconfig \
  --type=string
oc-west annotate secret istio-remote-secret-east-cluster -n istio-system networking.istio.io/cluster='east-cluster'
oc-west label secret istio-remote-secret-east-cluster -n istio-system istio/multiCluster='true'

oc-east create secret generic istio-remote-secret-west-cluster \
  -n istio-system \
  --from-file=west-cluster=$KUBECONFIG_LOCATION/istiod-basic-west-cluster.kubeconfig \
  --type=string
oc-east annotate secret istio-remote-secret-west-cluster -n istio-system networking.istio.io/cluster='west-cluster'
oc-east label secret istio-remote-secret-west-cluster -n istio-system istio/multiCluster='true'
```

7. Deploy bookinfo on west cluster and sleep on east cluster:
```shell
oc-west new-project bookinfo
oc-west label namespace bookinfo istio-injection=enabled
oc-west apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/platform/kube/bookinfo.yaml -n bookinfo
oc-east new-project sleep
oc-east label namespace sleep istio-injection=enabled
oc-east apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/sleep/sleep.yaml -n sleep
```

8. Update mesh networks:
```shell
# On AWS
WEST_HOSTNAME=$(oc-west get services istio-eastwestgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
# On GCP and Azure (and IBM?)
WEST_HOSTNAME=$(oc-west get services istio-eastwestgateway -n istio-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
sed -e "s/{{clusterNamePrefix}}/east/g" \
  -e "s/remote-network/west-network/" \
  -e "s/remote-cluster/west-cluster/" \
  -e "s/remote-gateway/$WEST_HOSTNAME/" smcp.tmpl.yaml | oc-east apply -n istio-system -f -
```

9. Test connectivity between services:
```shell
istioctl --kubeconfig=$KUBECONFIG_EAST pc endpoints $(oc-east get pods -n sleep -l app=sleep -o jsonpath='{.items[].metadata.name}') -n sleep
oc-east exec $(oc-east get pods -l app=sleep -n sleep -o jsonpath='{.items[].metadata.name}') -n sleep -c sleep -- \
  curl -v "productpage.bookinfo:9080/productpage"
```

#### Identified issues:
1. Istio Operator does not create service account `istio-reader-service-account` that should be used by remote cluster.
2. Ingress and egress gateways cannot be disabled when `spec.multi-cluster` is specified.
3. Istiod does not discover east-west gateway when `cluster.multiCluster.meshNetworks` is not specified.
4. TODO: Check if disabling federation will fix issue 3.
5. TODO: Try to create `meshNetworks` in a config map.
6. TODO: Enable Kiali.
7. TODO: Investigate why `registryServiceName` does not work.
