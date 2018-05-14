#!/bin/bash

# This script copies the certs to each microservice directory. Each microservice
# has its own dockerfile and you cannot add anything to a dockerfile that is
# outside its build path.

declare -a services=("lcm" "jobmonitor")
for i in ${services[@]}
do
  mkdir -p ../cmd/${i}/certs
  cp server.crt server.key ca.crt ../cmd/${i}/certs
done
