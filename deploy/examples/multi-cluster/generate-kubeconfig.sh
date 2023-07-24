#!/bin/bash

# Set these values by input arguments
clusterName='east-cluster'
server='https://<api-server-url>:6443'
namespace='istio-system'
serviceAccount='istiod-basic'
secretName='istiod-basic-token-zrf4d'

set -o errexit

ca=$(KUBECONFIG=<path-to-kubeconfig-file> oc -n "$namespace" get secret "$secretName" -o=jsonpath='{.data.ca\.crt}')
token=$(KUBECONFIG=<path-to-kubeconfig-file> oc -n "$namespace" get secret "$secretName" -o=jsonpath='{.data.token}' | base64 --decode)

echo "apiVersion: v1
kind: Config
clusters:
  - name: ${clusterName}
    cluster:
      certificate-authority-data: ${ca}
      server: ${server}
contexts:
  - name: ${serviceAccount}@${clusterName}
    context:
      cluster: ${clusterName}
      namespace: ${namespace}
      user: ${serviceAccount}
users:
  - name: ${serviceAccount}
    user:
      token: ${token}
current-context: ${serviceAccount}@${clusterName}"
