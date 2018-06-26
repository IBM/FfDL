#!/bin/bash

declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))
for node in "${arrNodes[@]}"
do
docker cp $FFDL_PATH/bin/ibmc-s3fs $node:/root/ibmc-s3fs
docker cp $FFDL_PATH/bin/s3fs $node:/usr/local/bin/s3fs
docker exec -i $node /bin/bash <<_EOF
apt-get -y update
apt-get -y install s3fs
mkdir -p /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs
cp /root/ibmc-s3fs /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs
chmod +x /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs/ibmc-s3fs
systemctl restart kubelet
_EOF
done
