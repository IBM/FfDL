#!/bin/bash
sudo apt-get update -qq
sudo apt-get install -y build-essential git-core libcurl4-openssl-dev libxml2-dev pkg-config autotools-dev automake libfuse-dev fuseliblibssl-dev
git clone https://github.com/s3fs-fuse/s3fs-fuse.git
cd s3fs-fuse
./autogen.sh
./configure CPPFLAGS='-I/usr/local/opt/openssl/include'
make