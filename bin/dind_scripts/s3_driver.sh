#!/bin/bash
mkdir -p $GOPATH/src/github.com/IBM
mkdir -p $GOPATH/bin
cd $GOPATH/src/github.com/IBM/
git clone https://github.com/IBM/ibmcloud-object-storage-plugin.git
cd ibmcloud-object-storage-plugin
make
make provisioner
make driver

kubectl create secret docker-registry regcred --docker-server=${DOCKER_REPO} --docker-username=${DOCKER_REPO_USER} --docker-password=${DOCKER_REPO_PASS} --docker-email=unknown@docker.io
docker tag ibmcloud-object-storage-plugin ${DOCKER_REPO}/ibmcloud-object-storage-plugin
docker push ${DOCKER_REPO}/ibmcloud-object-storage-plugin

declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))
for node in "${arrNodes[@]}"
do
docker cp $GOPATH/bin/ibmc-s3fs $node:/root/ibmc-s3fs
docker cp ~/s3fs-fuse/src/s3fs $node:/usr/local/bin/s3fs
docker exec -i $node /bin/bash <<_EOF
mkdir -p /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs
cp /root/ibmc-s3fs /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs
chmod +x /usr/libexec/kubernetes/kubelet-plugins/volume/exec/ibm~ibmc-s3fs/ibmc-s3fs
systemctl restart kubelet
docker pull $DOCKER_REPO/ibmcloud-object-storage-plugin:latest
_EOF
done

cd $GOPATH/src/github.com/IBM/ibmcloud-object-storage-plugin/
kubectl create -f deploy/provisioner-sa.yaml
cp deploy/provisioner.yaml deploy/provisioner_reg.yaml
sed -i "s/image: ibmcloud-object-storage-plugin:latest/image: $(hostname).$(dnsdomainname)\/ibmcloud-object-storage-plugin:latest/g" deploy/provisioner_reg.yaml
kubectl create -f deploy/provisioner_reg.yaml
kubectl create -f deploy/ibmc-s3fs-standard-StorageClass.yaml