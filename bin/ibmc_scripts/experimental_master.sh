#!/bin/bash
operating_system=$(uname)
if [[ "$operating_system" == 'Linux' ]]; then
    CMD_SED=sed
elif [[ "$operating_system" == 'Darwin' ]]; then
    CMD_SED=gsed
fi

# Please enter this information
export DOCKER_REPO_PASS=
export DOCKER_NAMESPACE=
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=
export PUBLIC_IP=
export CLUSTER_NAME=
# Potentially export KUBECONFIG=... to point kubectl to your cluster

# This information should be largely static
export DOCKER_REPO=registry.ng.bluemix.net
export DLAAS_USERID=user-$(whoami)
export DOCKER_REPO_USER=token
export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold"
export DLAAS_IMAGE_PULL_POLICY=Always
export DOCKER_PULL_POLICY=Always
export VM_TYPE=none
export HAS_STATIC_VOLUMES=True

# Build S3 Driver
cd $GOPATH/src/github.com/IBM/
git clone https://github.com/IBM/ibmcloud-object-storage-plugin.git
cd ibmcloud-object-storage-plugin/deploy/binary-build-and-deploy-scripts/
./build-all.sh

# Push to registry
docker login --username=${DOCKER_REPO_USER} --password=${DOCKER_REPO_PASS} https://${DOCKER_REPO}
docker tag ibmcloud-object-storage-deployer:v001 $DOCKER_REPO/$DOCKER_NAMESPACE/ibmcloud-object-storage-deployer:v001
docker tag ibmcloud-object-storage-plugin:v001 $DOCKER_REPO/$DOCKER_NAMESPACE/ibmcloud-object-storage-plugin:v001
docker push $DOCKER_REPO/$DOCKER_NAMESPACE/ibmcloud-object-storage-deployer:v001
docker push $DOCKER_REPO/$DOCKER_NAMESPACE/ibmcloud-object-storage-plugin:v001

# Deploy S3 Driver
# Replace image tag in yaml descriptors to point to registry and namespace
$CMD_SED -i "s/image: \"ibmcloud-object-storage-deployer:v001\"/image: \"$DOCKER_REPO\/$DOCKER_NAMESPACE\/ibmcloud-object-storage-deployer:v001\"/g" deploy-plugin.yaml
$CMD_SED -i "s/image: \"ibmcloud-object-storage-plugin:v001\"/image: \"$DOCKER_REPO\/$DOCKER_NAMESPACE\/ibmcloud-object-storage-plugin:v001\"/g" deploy-provisioner.yaml

# Create secret, then deploy daemonset and plugin
kubectl create secret docker-registry regcred --docker-server=${DOCKER_REPO} --docker-username=${DOCKER_REPO_USER} --docker-password=${DOCKER_REPO_PASS} --docker-email=unknown@docker.io -n kube-system
# Note: Running deploy-plugin daemonset might only be possible once (and is also only needed once).
#       Running it additional times might cause FfDL deployment to wait indefinitely. If that should happen, just delete it again.
kubectl create -f deploy-plugin.yaml
kubectl create -f deploy-provisioner.yaml

# Create storage class
cd ..
kubectl create -f ibmc-s3fs-standard-StorageClass.yaml

# Deploy FfDL
cd $GOPATH/src/github.com/IBM/FfDL/bin
./create_static_volumes.sh
./create_static_volumes_config.sh
cd ..
glide install
make docker-build-base
make build gen-certs docker-build docker-push deploy

echo ""
echo "Deployment complete. You can now run a test job with:"
echo "    make test-submit"