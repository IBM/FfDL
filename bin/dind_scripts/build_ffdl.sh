#!/bin/bash
mkdir -p $GOPATH/src/github.com/IBM && cd $_
git clone https://github.com/IBM/FfDL.git && cd FfDL
#git remote add upstream https://github.com/IBM/FfDL.git
#git fetch --all
#git checkout master && git pull upstream master
#git fetch upstream pull/79/head:pr-79
#git checkout pr-79
glide install
make build gen-certs docker-build-base