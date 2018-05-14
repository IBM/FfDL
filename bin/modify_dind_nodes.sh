#!/bin/bash
DOCKER_NAMESPACE="ffdl"
DOCKER_REPO=${DOCKER_REPO:-docker.io}
DOCKER_USER=${DOCKER_REPO_USER:-test}
DOCKER_PASS=${DOCKER_REPO_PASS:-test}

declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))

# Login to Docker Registry
#echo "docker login --username=$DOCKER_USER --password=$DOCKER_PASS https://$DOCKER_REPO"
#docker login --username=$DOCKER_USER --password=$DOCKER_PASS https://$DOCKER_REPO

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
apt-get install -y nfs-common
update-ca-certificates
service docker restart
exit
_EOF
done
