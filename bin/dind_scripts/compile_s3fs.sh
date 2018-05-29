#!/bin/bash
sudo apt-get update -qq
cd ~
sudo apt-get install -y build-essential git-core libcurl4-openssl-dev libxml2-dev pkg-config autotools-dev automake libfuse-dev libssl-dev
git clone https://github.com/s3fs-fuse/s3fs-fuse.git
cd s3fs-fuse
./autogen.sh
./configure CPPFLAGS='-I/usr/local/opt/openssl/include'
make

mkdir -p $GOPATH/src/github.com/IBM
mkdir -p $GOPATH/bin
cd $GOPATH/src/github.com/IBM/
git clone https://github.com/IBM/ibmcloud-object-storage-plugin.git
cd ibmcloud-object-storage-plugin
make
make provisioner
make driver