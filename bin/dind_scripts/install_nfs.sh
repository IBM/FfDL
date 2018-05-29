#!/bin/bash
sudo apt install -y nfs-kernel-server
sudo mkdir -p /nfs/var/nfs/general
sudo chown nobody:nogroup /nfs/var/nfs/general
echo "/nfs/var/nfs/general    *(rw,sync,no_root_squash,no_subtree_check)" | sudo tee -a /etc/exports
sudo systemctl restart nfs-kernel-server
echo "NFS installed. Please note that you need nfs-common on every Kubernetes node you wish to use NFS on."