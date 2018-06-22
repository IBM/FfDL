#!/bin/bash

# Create a configmap that has the spec of the static volumes.

# Arguments:
# $1: only use volumes with label "type: $1" (optional, defaults to "dlaas-static-volume")

#DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

CONFIGMAP_NAME=static-volumes
volumeType=${1:-dlaas-static-volume}

# Delete configmap
kubectl delete configmap ${CONFIGMAP_NAME}

# Create new configmap
echo
echo "Using volumes with label type=$volumeType:"
kubectl get pvc --selector type=${volumeType}
echo
kubectl create configmap ${CONFIGMAP_NAME} --from-file=PVCs.yaml=<(
    kubectl get pvc --selector type=${volumeType} -o yaml
)
