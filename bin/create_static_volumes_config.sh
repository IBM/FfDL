#!/bin/bash

# Create a configmap that has the spec of the static volumes.

# Arguments:
# $1: only use volumes with label "type: $1" (optional, defaults to "dlaas-static-volume")

#DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

CONFIGMAP_NAME=static-volumes
volumeType=${1:-dlaas-static-volume}
NAMESPACE=${NAMESPACE:-default}

# Delete configmap
if kubectl get cm -n ${NAMESPACE} | grep static-volumes &> /dev/null; then kubectl delete configmap ${CONFIGMAP_NAME} -n ${NAMESPACE}; else echo "No need to delete ${CONFIGMAP_NAME} since it doesn't exist."; fi

# Create new configmap
echo
echo "Using volumes with label type=$volumeType:"
kubectl get pvc --selector type=${volumeType} -n ${NAMESPACE}
echo
kubectl create configmap ${CONFIGMAP_NAME} -n ${NAMESPACE} --from-file=PVCs.yaml=<(
    kubectl get pvc --selector type=${volumeType} -n ${NAMESPACE} -o yaml
)

CONFIGMAP_NAME2=static-volumes-v2

# Delete configmap
#if kubectl get cm | grep static-volumes &> /dev/null; then kubectl delete configmap ${CONFIGMAP_NAME2}; else echo "No need to delete ${CONFIGMAP_NAME2} since it doesn't exist."; fi

# Create new configmap
echo
echo "Using volumes with label type=$volumeType"
kubectl get pvc --selector type=${volumeType} -n ${NAMESPACE}
echo

kubectl get pvc --selector type="dlaas-static-volume" -n ${NAMESPACE} -o jsonpath='{"static-volumes-v2:"}{range .items[*]}{"\n  - name: "}{.metadata.name}{"\n    zlabel: "}{.metadata.name}{"\n    status: active\n"}' > PVCs-v2.yaml

kubectl create configmap ${CONFIGMAP_NAME2} -n ${NAMESPACE} --from-file=PVCs-v2.yaml
