## Integration with cert-manager and istio-csr in multi-tenant environment

1. Install cert-manager
```shell
helm install cert-manager jetstack/cert-manager \
    --namespace cert-manager \
    --create-namespace \
    --version v1.11.0 \
    --set installCRDs=true
```

2. Provision certificates:
```shell
oc new-project istio-system-1
oc new-project istio-system-2
oc apply -f deploy/examples/cert-manager/multi-tenancy/selfsigned-ca.yaml
```

3. Deploy Istio and istio-csr (mesh 1):
```shell
helm install istio-csr-mesh-1 jetstack/cert-manager-istio-csr \
    -n istio-system-1 \
    -f deploy/examples/cert-manager/multi-tenancy/istio-csr-mesh-1.yaml
oc apply -f deploy/examples/cert-manager/multi-tenancy/mesh-1.yaml
```

4. Deploy bookinfo in mesh 1:
```shell
oc new-project bookinfo
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/platform/kube/bookinfo.yaml -n bookinfo
```

5. Deploy Istio and istio-csr (mesh 2):
```shell
helm install istio-csr-mesh-2 jetstack/cert-manager-istio-csr \
    -n istio-system-2 \
    -f deploy/examples/cert-manager/multi-tenancy/istio-csr-mesh-2.yaml
oc apply -f deploy/examples/cert-manager/multi-tenancy/mesh-2.yaml
```

6. Deploy Istio:
```shell
oc apply -f deploy/examples/cert-manager/smcp.yaml -n istio-system
```
