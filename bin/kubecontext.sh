#!/bin/bash

# Print selected information from a Kubernetes context.
# Be sure to set the DLAAS_KUBE_CONTEXT to override the current context (i.e., kubectl config current-context)

DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

function printServerCertificate() {
  cluster=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.cluster}')
  value=$(kubectl config view -o jsonpath='{.clusters[?(@.name=="'$cluster'")].cluster.certificate-authority}')
  echo $value
}

function printClientCertificate() {
  user=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.user}')
  value=$(kubectl config view -o jsonpath='{.users[?(@.name=="'$user'")].user.client-certificate}')
  echo $value
}

function printClientKey() {
  user=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.user}')
  value=$(kubectl config view -o jsonpath='{.users[?(@.name=="'$user'")].user.client-key}')
  echo $value
}

function printUserToken() {
  user=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.user}')
  token=$(kubectl config view -o jsonpath='{.users[?(@.name=="'$user'")].user.token}')
  echo $token
}

function printNamespace() {
  namespace=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.namespace}')
  if [ -z "$namespace" ]; then
    namespace="default"
  fi
  echo $namespace
}

function printApiServer() {
  cluster=$(kubectl config view -o jsonpath='{.contexts[?(@.name=="'$DLAAS_KUBE_CONTEXT'")].context.cluster}')
  server=$(kubectl config view -o jsonpath='{.clusters[?(@.name=="'$cluster'")].cluster.server}')
  echo $server
}

function printRestApiUrl() {
  host=$(kubectl --context="$DLAAS_KUBE_CONTEXT" get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
  port=$(kubectl --context="$DLAAS_KUBE_CONTEXT" get svc/ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}' 2> /dev/null)
  if [[ -z "$host" || -z "$port" ]]; then
    # Assume run-services-local scenario.
    url="localhost:10000"
  else
    url="$host:$port"
  fi
  echo $url
}

function printTrainerUrl() {
  host=$(kubectl --context="$DLAAS_KUBE_CONTEXT" get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
  port=$(kubectl --context="$DLAAS_KUBE_CONTEXT" get svc/ffdl-trainer-v2 -o jsonpath='{.spec.ports[0].nodePort}' 2> /dev/null)
  if [[ -z "$host" || -z "$port" ]]; then
    # Assume run-services-local scenario.
    url="localhost:30005"
  else
    url="$host:$port"
  fi
  echo $url
}


# Parse command line args and dispatch command.
case $1 in
server-certificate)
  printServerCertificate
  ;;
client-certificate)
  printClientCertificate
  ;;
client-key)
  printClientKey
  ;;
user-token)
  printUserToken
  ;;
namespace)
  printNamespace
  ;;
api-server)
  printApiServer
  ;;
restapi-url)
  printRestApiUrl
  ;;
trainer-url)
  printTrainerUrl
  ;;
*)
  echo "Usage: kubecontext.sh [server-certificate|client-certificate|client-key|user-token|namespace|api-server|restapi-url|trainer-url]"
  exit 1
  ;;
esac
