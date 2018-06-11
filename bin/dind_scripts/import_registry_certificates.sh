#!/bin/bash
declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))

# Login to Docker Registry
#echo "docker login --username=$DOCKER_USER --password=$DOCKER_PASS https://$DOCKER_REPO"
#docker login --username=$DOCKER_USER --password=$DOCKER_PASS https://$DOCKER_REPO

docker login -u="$USER" -p="7312mInalM4n" $(hostname).$(dnsdomainname)
DOCKER_AUTH=$(cat ~/.docker/config.json | jq ".auths.\"$DOCKER_REPO\".auth")
echo $DOCKER_AUTH
cat <<_EOF > config.json
{
  "auths": {
    "$DOCKER_REPO": {
      "auth": $DOCKER_AUTH
    }
  }
}
_EOF

for node in "${arrNodes[@]}"
do
   docker exec -d $node mkdir -p /usr/local/share/ca-certificates/docker-dev-cert/
   echo "docker cp ~/docker-registry/nginx/devdockerCA.crt $node:/usr/local/share/ca-certificates/docker-dev-cert/devdockerCA.crt"
   docker cp ~/docker-registry/nginx/devdockerCA.crt $node:/usr/local/share/ca-certificates/docker-dev-cert/devdockerCA.crt
   docker exec -d $node mkdir -p /root/.docker/
   docker cp config.json $node:/root/.docker/config.json
   docker exec -i $node /bin/bash <<'_EOF'
apt-get update && apt-get install -y libssl1.0.2 nfs-common libfuse2 libxml2 curl libcurl3
update-ca-certificates
service docker restart
exit
_EOF
done
