1. Deploy SMCP and create SMMR:
```shell
oc new-project istio-system
oc apply -f mesh.yaml -n istio-system
```

2. Create fake 3scale system:
```shell
oc new-project 3scale
oc apply -f 3scale-system.yaml -n 3scale
```

3. Apply 3scale plugin to the ingress gateway:
```shell
oc apply -f wasm-plugin-ingress-gateway.yaml
```

4. Configure authorization:
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
oc apply -f - <<EOF
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
  namespace: istio-system
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  action: ALLOW
  rules:
  - from:
    - source:
       requestPrincipals: ["testing@secure.istio.io/testing@secure.istio.io"]
EOF
```

5. Execute request:
```shell
TOKEN=$(curl https://raw.githubusercontent.com/istio/istio/release-1.19/security/tools/jwt/samples/demo.jwt -s) && echo "$TOKEN" | cut -d '.' -f2 - | base64 --decode -
curl -v -H "Authorization: Bearer $TOKEN" -H "Host: $ROUTE" -H "user_key: api-key" -H "app_key: app_key" "http://$ROUTE:80/productpage" > /dev/null
```

Notes:
The above request will fail, because it cannot connect to the 3scale backend,
but it communicates with 3scale admin API and computes credentials correctly.

Next steps:
Implement a mock for 3scale backend.

Hints:
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