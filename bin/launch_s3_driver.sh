#!/usr/bin/env bash
# Experimental(!) script to (re)launch S3 driver when it is already installed.

cd $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-addendum/

declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))
for node in "${arrNodes[@]}"
do
docker cp ibmcloud-object-storage-driver.tar $node:/
docker cp ibmcloud-object-storage-plugin.tar $node:/
docker exec -i $node /bin/bash <<'_EOF'
docker load --input ibmcloud-object-storage-driver.tar
docker load --input ibmcloud-object-storage-plugin.tar
apt install -y libfuse2 libxml2
_EOF
done

kubectl create configmap -n  kube-system cluster-info --from-file=cluster-config.json=./dummy.cm
cd $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-plugin/deploy/helm
helm init --wait
helm install ./ibmcloud-object-storage-plugin-security
helm install ./ibmcloud-object-storage-plugin
# Delete storage plugin to restart it
kubectl delete $(kubectl get pods -n kube-system -o name | grep ibmcloud-object-storage-plugin-) -n kube-system
echo "Basic S3 driver deployment complete. Storage Information:"
kubectl get storageclass | grep s3
echo "-------"
kubectl get pods -n kube-system | grep object-storage

cd $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-addendum/
kubectl apply -f secret.yaml
kubectl apply -f pvc.yaml
kubectl apply -f pod.yaml

echo "Please wait until the pods are up and then test as follows:"
echo "kubectl exec -it s3fs-test-pod bash"
echo "    df -Th | grep s3"
echo "    ls /mnt/s3fs/"
