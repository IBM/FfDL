#!/bin/bash

# Create a pod that mounts the static volumes.
# This is a debugging convenience.

DLAAS_KUBE_CONTEXT=${DLAAS_KUBE_CONTEXT:-$(kubectl config current-context)}

# Get list of persistent volume claims with the right labels
#volumeNames=$(kubectl --context ${DLAAS_KUBE_CONTEXT} get pvc --selector type=dlaas-static-volume -o jsonpath='{.items[*].metadata.name}')
volumeNames=$(kubectl --context ${DLAAS_KUBE_CONTEXT} get pvc -o jsonpath='{.items[*].metadata.name}')

# Pod spec template.
podspec=$(cat <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: static-volumes-mountall
spec:
  containers:
  - image: busybox
    name: mountall
    command: ["sleep", "3600"]
    volumeMounts:
    resources:
      limits:
        cpu: 1
        memory: 512Mi
  volumes:
EOF
)

# Update pod spec to include the static volumes.
i=0
for volumeName in $volumeNames; do
  echo Found volume: $volumeName

  # Add volumes to spec.
  podspec=$(echo "$podspec" \
    |yq w - "spec.volumes[$i].name" "$volumeName" \
    |yq w - "spec.volumes[$i].persistentVolumeClaim[claimName]" "$volumeName"
  )

  # Add mounts to spec.
  podspec=$(echo "$podspec" \
    |yq w - "spec.containers[0].volumeMounts[$i].name" "$volumeName" \
    |yq w - "spec.containers[0].volumeMounts[$i].mountPath" "/mnt/$volumeName"
  )

  i=$((i+1))
done

echo "$podspec" | kubectl --context "$DLAAS_KUBE_CONTEXT" create -f -

