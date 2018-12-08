#!/bin/bash
if ! [ -x "$(command -v kubectl)" ]; then
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
    chmod +x ./kubectl
    sudo mv ./kubectl /usr/local/bin/kubectl
else
   echo "kubectl already installed."
fi

if ! [ -x "$(command -v helm)" ]; then
wget https://storage.googleapis.com/kubernetes-helm/helm-v2.8.2-linux-amd64.tar.gz
tar -zxvf helm-v2.8.2-linux-amd64.tar.gz
sudo mv linux-amd64/helm /usr/local/bin/
else
   echo "helm already installed."
fi

if [ ! -f ~/dind-cluster-v1.9.sh ]; then
    cd ~
    wget https://cdn.jsdelivr.net/gh/Mirantis/kubeadm-dind-cluster/fixed/dind-cluster-v1.9.sh
    chmod +x dind-cluster-v1.9.sh
else
    echo "DIND Kubernetes script already exists. Nothing to do."
fi