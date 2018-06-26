#!/usr/bin/env bash
DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

echo "Creating persistent volume."

(kubectl --context "$DLAAS_KUBE_CONTEXT" create -f - <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: nfs
spec:
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteMany
  nfs:
    server: ${DOCKER_REPO}
    path: "/nfs/var/nfs/general"
EOF
) || true
