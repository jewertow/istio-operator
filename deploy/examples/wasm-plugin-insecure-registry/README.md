## Pulling WASM plugin image from insecure registry

### Prerequisites:

You need to install the following operators:
1. OpenShift Data Foundation
2. Quay
3. Service Mesh

### Steps

1. Configure object storage for Quay:
```shell
oc apply -f - <<EOF
apiVersion: noobaa.io/v1alpha1
kind: NooBaa
metadata:
  name: noobaa
  namespace: openshift-storage
spec:
 dbResources:
   requests:
     cpu: '0.1'
     memory: 1Gi
 dbType: postgres
 coreResources:
   requests:
     cpu: '0.1'
     memory: 1Gi
EOF
```

2. Deploy Quay registry:
```shell
oc new-project quay
oc apply -f - <<EOF
apiVersion: quay.redhat.com/v1
kind: QuayRegistry
metadata:
  name: test-registry
  namespace: quay
spec:
  components:
    - kind: clair
      managed: false
    - kind: postgres
      managed: true
    - kind: objectstorage
      managed: true
    - kind: redis
      managed: true
    - kind: horizontalpodautoscaler
      managed: false
    - kind: route
      managed: true
    - kind: mirror
      managed: false
    - kind: monitoring
      managed: false
    - kind: tls
      managed: true
    - kind: quay
      managed: true
      overrides:
        replicas: 1
    - kind: clairpostgres
      managed: false
EOF
```

3. Fetch the registry endpoint:
```shell
REGISTRY_ADDR=$(oc get quayregistry test-registry -o jsonpath='{.status.registryEndpoint}')
echo $REGISTRY_ADDR
REGISTRY=$(basename $REGISTRY_ADDR)
```

4. Open registry in a browser and create `admin` user.

5. Generate and download pull secret: go to **Account settings** -> **Generate Encrypted Password** -> **Kubernetes Secret**.

6. Configure your docker client to trust the test registry - on Linux,
you can add the following config to `/etc/docker/daemon.json` and restart docker service:
```json
{
    "insecure-registries" : ["<REGISTRY>"]
}
```
```shell
sudo systemctl restart docker
```

7. Log in to the test registry and push a WASM plugin:
```shell
docker login $REGISTRY
docker pull quay.io/3scale/threescale-wasm-auth:0.0.4
docker tag quay.io/3scale/threescale-wasm-auth:0.0.4 $REGISTRY/admin/trheescale-wasm-auth:0.0.4
docker push $REGISTRY/admin/trheescale-wasm-auth:0.0.4
```

7. Deploy Service Mesh control plane:
```shell
oc new-project istio-system
oc apply -f - <<EOF
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: basic
  namespace: istio-system
spec:
  addons:
    kiali:
      enabled: false
    prometheus:
      enabled: false
    grafana:
      enabled: false
  gateways:
    egress:
      enabled: false
    openshiftRoute:
      enabled: false
  general:
    logging:
      componentLevels:
        default: info
  proxy:
    accessLogging:
      file:
        name: /dev/stdout
  tracing:
    type: None
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - bookinfo
EOF
oc new-project bookinfo
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/platform/kube/bookinfo.yaml -n bookinfo
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/networking/bookinfo-gateway.yaml -n bookinfo
```

8. Configure proxies to allow pulling WASM plugins from insecure registry:
```shell
echo "apiVersion: networking.istio.io/v1beta1
kind: ProxyConfig
metadata: 
  name: enable-insecure-registry
  namespace: istio-system
spec: 
  environmentVariables: 
    WASM_INSECURE_REGISTRIES: \"$REGISTRY\"
" | oc apply -f -
```

9. Apply WASM plugin to the istio-ingressgateway:
```shell
oc apply -f admin-secret.yml -n istio-system

echo "apiVersion: extensions.istio.io/v1alpha1
kind: WasmPlugin
metadata:
  name: 3scale-plugin
  namespace: istio-system
spec:
  url: oci://$REGISTRY/admin/threescale-wasm-auth:0.0.4
  phase: AUTHZ
  match:
  - mode: SERVER
  imagePullPolicy: Always
  imagePullSecret: admin-pull-secret
  pluginConfig:
    api: v1
    system:
      name: 3scale-system
      upstream:
        name: outbound|80||system.3scale.svc.cluster.local
        url: http://system.3scale.svc.cluster.local
        timeout: 5000
      token: abc
    backend:
      name: 3scale-backend
      upstream:
        name: outbound|80||backend.3scale.svc.cluster.local
        url: http://backend.3scale.svc.cluster.local
        timeout: 5000
      extensions:
      - no_body
    services:
    - id: '123'
      authorities:
      - '*'
      credentials:
        user_key:
        - filter:
            path:
            - envoy.filters.http.jwt_authn
            - '0'
            keys:
            - foo
            ops:
            - take:
                head: 1
" | oc apply -f -
```
