#!/bin/bash

# Helper script to create static volumes.

# SCRIPTDIR="$(cd $(dirname "$0")/ && pwd)"

# Be sure to set the DLAAS_KUBE_CONTEXT to override the current context (i.e., kubectl config current-context)
DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

#echo "Kube context: $DLAAS_KUBE_CONTEXT"

# Should be "ibmc-file-gold" for Bluemix deployment
SHARED_VOLUME_STORAGE_CLASS="${SHARED_VOLUME_STORAGE_CLASS:-""}"

volumeNum=${1:-1}

echo "Creating persistent volume claim $volumeNum"
(kubectl --context "$DLAAS_KUBE_CONTEXT" create -f - <<EOF
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: static-volume-$volumeNum
  annotations:
    volume.beta.kubernetes.io/storage-class: "$SHARED_VOLUME_STORAGE_CLASS"
  labels:
    type: dlaas-static-volume
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 20Gi
EOF
) || true

