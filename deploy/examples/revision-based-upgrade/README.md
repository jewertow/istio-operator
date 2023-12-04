### Revision-based upgrade from OSSM 2.4 to Istio 1.20

#### Prerequisites
1. Install OSSM Operator.
2. Install `istioctl` 1.20.
3. Optional: enable OpenShift monitoring stack.

#### Steps
1. Enable monitoring for user-defined projects:
```shell
oc apply -f enable-monitoring.yaml
```
2. Deploy SMCP:
```shell
oc create namespace istio-system
oc apply -f smcp.yaml -n istio-system
oc apply -f route.yaml -n istio-system
oc apply -f istio-monitor.yaml -n istio-system
oc apply -f proxy-monitor.yaml -n istio-system
```

4. Wait until Istio is ready and then enable telemetry:
```shell
oc apply -f telemetry.yaml -n istio-system
```

5. Deploy Bookinfo:
```shell
oc new-project bookinfo
oc apply -f proxy-monitor.yaml -n bookinfo
# This label is used for discovery
oc label namespace bookinfo mesh=true
# This label is used for injection
oc label namespace bookinfo istio.io/rev=prod-stable
oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/platform/kube/bookinfo.yaml -n bookinfo
# TODO: investigate why Gateway with port 80 does not work
# oc apply -f https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/bookinfo/networking/bookinfo-gateway.yaml -n bookinfo
oc apply -f bookinfo-gateway.yaml -n bookinfo
```

6. Generate ingress traffic:
```shell
while true; do curl -v http://bookinfo.apps-crc.testing:80/productpage > /dev/null; sleep 1; done
```

7. Deploy new control plane:
```shell
istioctl install -y -f istio-1-20-canary.yaml -n istio-system
istioctl tag set prod-canary --revision 1-20
oc apply -f istio-ingressgateway-1-20-canary.yaml -n istio-system
oc apply -f istio-ingressgateway-1-20-svc.yaml -n istio-system
# wait until gateway is created and check if gateway config was sent
INGRESS_GW_NAME=$(oc get pods -l app=istio-ingressgateway-1-20-canary -n istio-system -o jsonpath='{.items[0].metadata.name}')
istioctl pc listeners $INGRESS_GW_NAME -n istio-system
kubectl apply -f mtls.yaml -n istio-system
```

8. Switch traffic to the new gateway:
```shell
oc patch route istio-ingressgateway -n istio-system --patch '{"spec": {"to": {"name": "istio-ingressgateway-1-20"}}}'
```

9. Switch productpage app to the new revision:
```shell
kubectl label namespace bookinfo istio.io/rev=prod-canary --overwrite
kubectl rollout restart deployment productpage-v1 -n bookinfo
```

Istio proxy in productpage pod should log that it connects to istio-1-20.istio-system.

10. Disable injection in SMCP:
```shell
kubectl apply -f disable-smmr.yaml -n istio-system
```

11. Switch bookinfo to the new revision:
```shell
kubectl label namespace bookinfo istio.io/rev=prod-stable --overwrite
istioctl tag set prod-stable --revision 1-20 --overwrite
kubectl get deployments -n bookinfo | awk '{print $1}' | awk '!/NAME/' | xargs kubectl -n bookinfo rollout restart deployment
```

12. Update revision in the new ingress gateway:
```shell
kubectl apply -f istio-ingressgateway-1-20.yaml -n istio-system
```
Wait until the new pod is ready and remove the previous deployment:
```shell
kubectl delete -f istio-ingressgateway-1-20-canary.yaml -n istio-system
```

13. Delete SMCP and redeploy Istio with enabled validation webhook:
```shell
oc delete -f smcp.yaml -n istio-system
oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io
oc delete mutatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io
oc delete svc maistra-admission-controller -n openshift-operators
oc delete ds -l maistra-version -n openshift-operators
oc delete clusterrole/istio-admin clusterrole/istio-cni clusterrolebinding/istio-cni
oc delete clusterrole istio-view istio-edit
oc delete cm -n openshift-operators maistra-operator-cabundle
oc delete cm -n openshift-operators istio-cni-config-v2-4
oc delete cm -n openshift-operators ior-leader
oc delete cm -n openshift-operators istio-namespace-controller-election
oc delete cm -n openshift-operators servicemesh-federation
```
```shell
istioctl install -y -f istio-1-20.yaml -n istio-system
# Recreate mutation webhook for tag prod-stable
istioctl tag set prod-stable --revision 1-20
```

#### Identified issues:
1. Istio or SMCP ignores workload label `istio.io/rev` when the namespace is labeled with this label as well.
2. Why did I have to change service and gateway ports from 80 to 8080 in istio-ingressgateway?
