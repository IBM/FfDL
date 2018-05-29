#!/bin/bash
sudo apt install -y jq
cd ~
sudo ./dind-cluster-v1.9.sh clean
sudo ./dind-cluster-v1.9.sh up
# cd ~/go/src/github.com/IBM/FfDL/bin
# chmod +x modify_dind_nodes.sh
# ./modify_dind_nodes.sh