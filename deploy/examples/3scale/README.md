### Case 1: 3scale plugin configured only in an ingress gateway

1. Deploy SMCP and create SMMR:
```shell
oc new-project istio-system
oc apply -f mesh.yaml -n istio-system
```

2. Create fake 3scale system:
```shell
oc new-project 3scale
oc apply -f 3scale-system.yaml
oc apply -f 3scale-backend.yaml
```

3. Deploy httpbin and configure its gateway:
```shell
oc new-project bookinfo
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin.yaml -n bookinfo
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml -n bookinfo
```

4. Configure JWT authentication on the ingress gateway:
```shell
oc apply -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-config
  namespace: istio-system
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  jwtRules:
  - issuer: "testing@secure.istio.io"
    jwksUri: "https://raw.githubusercontent.com/istio/istio/release-1.19/security/tools/jwt/samples/jwks.json"
    # it must be set to true, otherwise, 3scale plugin will not find "authorization" header
    forwardOriginalToken: true
EOF
```

5. Apply 3scale plugin to the ingress gateway:
```shell
sed "s/{{ .AppLabel }}/istio-ingressgateway/g" wasm-plugin.yaml | oc apply -n istio-system -f -
```

6. Send a request:
```shell
TOKEN=$(curl https://raw.githubusercontent.com/istio/istio/release-1.19/security/tools/jwt/samples/demo.jwt -s) && echo "$TOKEN" | cut -d '.' -f2 - | base64 --decode -
ROUTE=$(oc get routes -n istio-system istio-ingressgateway -o jsonpath='{.spec.host}')
curl -v "http://$ROUTE:80/headers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Host: $ROUTE" > /dev/null
```

### Case 2: 3scale plugin configured in a server app 

1. Enable JWT authentication and 3scale plugin in `httpbin` app:
```shell
sed "s/{{ .AppLabel }}/httpbin/g" request-auth.yaml | oc apply -n bookinfo -f -
sed "s/{{ .AppLabel }}/httpbin/g" wasm-plugin.yaml | oc apply -n bookinfo -f -
```
2. Deploy sleep:
```shell
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/sleep/sleep.yaml -n bookinfo
```

3. Send a request from sleep to httpbin:
```shell
SLEEP_POD=$(kubectl get pods -n bookinfo -l app=sleep -o jsonpath='{.items[].metadata.name}')
kubectl exec $SLEEP_POD -n bookinfo -c sleep -- curl -v -H "Authorization: Bearer $TOKEN" http://httpbin:8000/headers > /dev/null
```
The request should succeed, because 3scale plugin is not applied to `sleep` app.

4. Send a request to the ingress gateway:
```shell
TOKEN=$(curl https://raw.githubusercontent.com/istio/istio/release-1.19/security/tools/jwt/samples/demo.jwt -s) && echo "$TOKEN" | cut -d '.' -f2 - | base64 --decode -
ROUTE=$(oc get routes -n istio-system istio-ingressgateway -o jsonpath='{.spec.host}')
curl -v "http://$ROUTE:80/headers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Host: $ROUTE" > /dev/null
```
The request should succeed, because the JWT configuration includes `forwardOriginalToken: true`,
so the `Authorization` header is forwarded to httpbin. Otherwise, authentication would fail in httpbin.

### Case 3: 3scale plugin configured in a client app (in an outbound listener):
1. Enable JWT authentication and 3scale plugin in `sleep` app:
```shell
sed "s/{{ .AppLabel }}/sleep/g" request-auth.yaml | oc apply -n bookinfo -f -
sed "s/{{ .AppLabel }}/sleep/g" wasm-plugin.yaml | oc apply -n bookinfo -f -
```
This step does not make sense for `sleep`, because it does not expose any endpoint,
but it shows that 3scale plugin is applied to outbound listeners what will cause failures,
because JWT auth filter is applied only to inbound listeners.

3. Send a request from sleep to httpbin:
```shell
SLEEP_POD=$(kubectl get pods -n bookinfo -l app=sleep -o jsonpath='{.items[].metadata.name}')
kubectl exec $SLEEP_POD -n bookinfo -c sleep -- curl -v -H "Authorization: Bearer $TOKEN" http://httpbin:8000/headers > /dev/null
```
This request should return 403, because 3scale plugin is applied to outbound listener.

### Notes
To enable Envoy debug log for WASM plugin in the default ingress gateway,
it is required to modify istio-ingressgateway arguments as follows:
```yaml
          args:
            - proxy
            - router
            - '--domain'
            - $(POD_NAMESPACE).svc.cluster.local
            - '--proxyLogLevel=warning'
            - '--proxyComponentLogLevel=misc:error,wasm:debug'
            - '--log_output_level=default:info,wasm:debug'
```

```yaml
  selector:
    # TODO: It must be fixed in the docs
    matchLabels:
      istio: ingressgateway
  pluginConfig:
    api: v1
    system:
      name: 3scale-system
      upstream:
        # This is only used to set header 'x-3scale-cluster-name'. Couldn't it be removed?
        # Additionally, why does it need Istio notation and port, if the client uses default https port?
        # Could that even work for port 8443 or whatever else?
        # I don't think so, because the client only uses `upstream.url` and does not extract port from the name.
        #
        # This is also important to note that SMCP must have set outboundTrafficPolicy: REGISTRY_ONLY.
        # Otherwise, WASM plugin will not find upstream.name and will return BadArgument.
        #
        # TODO: 3scale system must expose HTTP endpoint:
        # HTTP/1.1 GET /admin/api/services/{service_id}/proxy/configs/production/latest.json?access_token={system.token}
        name: outbound|80||system.3scale.svc.cluster.local
        # Why is it needed?
        url: http://system.3scale.svc.cluster.local
        timeout: 5000
      token: abc
```