#!/bin/bash

if [ -d "/home/$USER/docker-registry" ]; then
    echo "Registry seems to exist. Nothing to do."
    exit 0
fi

sudo apt-get -y install apache2-utils

mkdir ~/docker-registry && cd $_
mkdir data
mkdir nginx

SUBJECT="/C=US/ST=NY/L=Armonk/O=IBM/CN=$(hostname).$(dnsdomainname)"
VALIDITY=365

cat > docker-compose.yml <<'_EOF'
registry:
  image: registry:2
  ports:
    - 127.0.0.1:5000:5000
  environment:
    REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY: /data
  volumes:
    - ./data:/data
nginx:
  image: "nginx:1.9"
  ports:
    - 443:443
  links:
    - registry:registry
  volumes:
    - ./nginx/:/etc/nginx/conf.d:ro
_EOF

cat > nginx/registry.conf <<'_EOF'
upstream docker-registry {
  server registry:5000;
}

server {
  listen 443;
  server_name myregistrydomain.com;

  # SSL
  ssl on;
  ssl_certificate /etc/nginx/conf.d/domain.crt;
  ssl_certificate_key /etc/nginx/conf.d/domain.key;

  # disable any limits to avoid HTTP 413 for large image uploads
  client_max_body_size 0;

  # required to avoid HTTP 411: see Issue #1486 (https://github.com/docker/docker/issues/1486)
  chunked_transfer_encoding on;

  location /v2/ {
    # Do not allow connections from docker 1.5 and earlier
    # docker pre-1.6.0 did not properly set the user agent on ping, catch "Go *" user agents
    if ($http_user_agent ~ "^(docker\/1\.(3|4|5(?!\.[0-9]-dev))|Go ).*$" ) {
      return 404;
    }

    # To add basic authentication to v2 use auth_basic setting plus add_header
    auth_basic "registry.localhost";
    auth_basic_user_file /etc/nginx/conf.d/registry.password;
    add_header 'Docker-Distribution-Api-Version' 'registry/2.0' always;

    proxy_pass                          http://docker-registry;
    proxy_set_header  Host              $http_host;   # required for docker client's sake
    proxy_set_header  X-Real-IP         $remote_addr; # pass on real client's IP
    proxy_set_header  X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header  X-Forwarded-Proto $scheme;
    proxy_read_timeout                  900;
  }
}
_EOF

cd nginx/
# echo "Enter Registry Password:"
# htpasswd -c registry.password $USER
htpasswd -b -c registry.password $USER 7312mInalM4n

openssl genrsa -passout pass:d123gOn1337h -out devdockerCA.key 2048
openssl req -x509 -new -nodes -days ${VALIDITY} -passin pass:d123gOn1337h -key devdockerCA.key -out devdockerCA.crt -subj ${SUBJECT}

openssl genrsa -passout pass:d123gOn1337h -out domain.key 2048
# NOTE: Enter your domain or IP as common name, e.g. name.sl.cloud9.ibm.com or 9.xxx.xxx.xxx !!!
openssl req -new -key domain.key -passin pass:d123gOn1337h -out dev-docker-registry.com.csr -subj ${SUBJECT}
# Self-sign certificate
openssl x509 -req -days ${VALIDITY} -in dev-docker-registry.com.csr -CA devdockerCA.crt -CAkey devdockerCA.key -CAcreateserial -passin pass:d123gOn1337h -out domain.crt

sudo mkdir /usr/local/share/ca-certificates/docker-dev-cert
sudo cp devdockerCA.crt /usr/local/share/ca-certificates/docker-dev-cert
sudo update-ca-certificates

echo "Registry installed."