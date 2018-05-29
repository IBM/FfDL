#!/bin/bash
cd ~/docker-registry && docker-compose up -d
echo "Registry started. Please use \"curl https://$USER:7312mInalM4n@$(hostname).$(dnsdomainname)/v2/_catalog\" to list available images."