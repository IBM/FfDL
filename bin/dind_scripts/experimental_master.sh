#!/bin/bash
chmod +x build_ffdl.sh create_user.sh import_registry_certificates.sh install_docker.sh install_go.sh install_kubernetes.sh install_nfs.sh install_registry.sh launch_kubernetes.sh launch_registry.sh s3_driver.sh
./create_user.sh
./install_docker.sh
. install_go.sh
./install_kubernetes.sh
./install_registry.sh
./install_nfs.sh
./build_ffdl.sh

export DOCKER_REPO=$(hostname).$(dnsdomainname)
export DOCKER_REPO_USER=$USER
export DOCKER_REPO_PASS=7312mInalM4n
export VM_TYPE=none
./launch_registry.sh
./launch_kubernetes.sh
./import_registry_certificates.sh
./s3_driver.sh

export DLAAS_IMAGE_PULL_POLICY=Always
export DOCKER_PULL_POLICY=Always
export HAS_STATIC_VOLUMES=True
export PUBLIC_IP=10.192.0.3
export SHARED_VOLUME_STORAGE_CLASS=""
cd $GOPATH/src/github.com/IBM/FfDL
make docker-build docker-push create-volumes deploy

echo "\nFfDL installed."
echo "You can now define your run a test job like this:"
echo "    export AWS_ACCESS_KEY_ID=..."
echo "    export AWS_SECRET_ACCESS_KEY=..."
echo "    make test-submit"