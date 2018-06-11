#!/bin/bash
sudo apt-get update -qq
#cd ~
sudo apt-get install -y build-essential git-core libcurl4-openssl-dev libxml2-dev pkg-config autotools-dev automake libfuse-dev libssl-dev
#git clone https://github.com/s3fs-fuse/s3fs-fuse.git
#cd s3fs-fuse
#./autogen.sh
#./configure CPPFLAGS='-I/usr/local/opt/openssl/include'
#make

# Compile s3fs and ibmc-s3fs binaries
docker build -t larva -f Dockerfile.compile .
docker run -d --name s3compiler -v /var/run/docker.sock:/var/run/docker.sock -v $(which docker):/usr/bin/docker larva tail -f /dev/null
#docker exec -i s3compiler /bin/bash <<_EOF
#cd /root/go/src/github.com/IBM/ibmcloud-object-storage-plugin/
#make provisioner
#make driver
#_EOF
#mkdir -p binary_executables
mkdir -p ~/s3fs-fuse/src/
docker cp s3compiler:/s3fs-fuse/src/s3fs ~/s3fs-fuse/src/s3fs
#docker cp s3compiler:/root/go/bin/ibmc-s3fs binary_executables/ibmc-s3fs
docker stop s3compiler && docker rm s3compiler
# Could save to container, i.e. docker commit s3compiler larva, or discard image, i.e. docker rmi larva

# Push all images (ibmcloud-object-storage-plugin, deployer, optionally provisioner-builder and driver-builder) to docker registry
#docker login --username=${DOCKER_REPO_USER} --password=${DOCKER_REPO_PASS} https://${DOCKER_REPO}
#docker tag ibmcloud-object-storage-plugin:latest ${DOCKER_REPO}/ibmcloud-object-storage-plugin:latest
#docker tag deployer:latest ${DOCKER_REPO}/deployer:latest
#docker push ${DOCKER_REPO}/ibmcloud-object-storage-plugin:latest
#docker push ${DOCKER_REPO}/deployer:latest

mkdir -p $GOPATH/src/github.com/IBM
mkdir -p $GOPATH/bin
cd $GOPATH/src/github.com/IBM/
git clone https://github.com/IBM/ibmcloud-object-storage-plugin.git
cd ibmcloud-object-storage-plugin
make
make provisioner
make driver
