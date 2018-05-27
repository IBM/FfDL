#!/bin/bash
apt update && apt upgrade -y && apt install -y build-essential curl wget
useradd -m ffdlr
groupadd docker
usermod -aG docker ffdlr
usermod -aG sudo ffdlr
if [ "$(dnsdomainname)" == "sl.cloud9.ibm.com" ]; then echo "170.225.15.112 public.dhe.ibm.com" >> /etc/hosts else echo "You do not seem to be running on SoftLayer, not modifying hosts file."; fi
passwd ffdlr
su - ffdlr