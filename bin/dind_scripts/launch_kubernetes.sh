#!/bin/bash
sudo apt install -y jq
cd ~

wget https://github.com/kubernetes-sigs/kubeadm-dind-cluster/releases/download/v0.1.0/dind-cluster-v1.13.sh
chmod +x dind-cluster-v1.13.sh

sudo ./dind-cluster-v1.13.sh clean
sudo ./dind-cluster-v1.13.sh up
# cd ~/go/src/github.com/IBM/FfDL/bin
# chmod +x modify_dind_nodes.sh
# ./modify_dind_nodes.sh
