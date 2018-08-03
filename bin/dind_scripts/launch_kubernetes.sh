#!/bin/bash
sudo apt install -y jq
cd ~

wget https://cdn.rawgit.com/kubernetes-sigs/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.9.sh
chmod +x dind-cluster-v1.9.sh

sudo ./dind-cluster-v1.9.sh clean
sudo ./dind-cluster-v1.9.sh up
# cd ~/go/src/github.com/IBM/FfDL/bin
# chmod +x modify_dind_nodes.sh
# ./modify_dind_nodes.sh