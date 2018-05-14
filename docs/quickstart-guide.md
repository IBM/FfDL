# Quick Start Guide for FfDL

*NOTE: THIS GUIDE IS INCOMPLETE AND EXPERIMENTAL. FOLLOW AT YOUR OWN RISK*

This quick start guide for FfDL describes how to setup FfDL on a baremetal instance.
Starting from setting up a user, Docker and Kubernetes we will setup S3 drivers, build and install FfDL and finally run an exemplary training job. 

## Kick Start on IBM Cloud

### Set environment
```
kubectl create secret docker-registry regcred --docker-username <registry_username> --docker-password <registry_password> --docker-server registry.ng.bluemix.net --docker-email <registry_email>
export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold"
export PUBLIC_IP=<IP_TO_CLUSTER>
export DOCKER_REPO_USER=<REPOSITORY_USER>
export DLAAS_IMAGE_PULL_POLICY=Always
export DOCKER_PULL_POLICY=Always
export DOCKER_REPO_PASS=<PASSWORD_TO_YOUR_REPOSITORY>
export DOCKER_NAMESPACE=<NAMESPACE_ON_IBM_CLOUD>
export DOCKER_REPO=registry.ng.bluemix.net
export CLUSTER_NAME=<IBM_CLOUD_CLUSTER_NAME>
export VM_TYPE=none
export HAS_STATIC_VOLUMES=True
```

### Build
```
glide install
make build gen-certs docker-build docker-push
```

### Deploy & Test
```
cd bin
./create_static_volumes.sh
./create_static_volumes_config.sh
# Wait while kubectl get pvc shows static-volume-1 in state Pending
cd ..
make deploy
make test-submit
```

## Kick Start on DIND

If you have no Kubernetes cluster, yet, please start with the next section.

```
# Start Kubernetes Cluster
./dind-cluster-v1.9.sh up
# Start Docker Registry
cd ~/docker-registry
docker-compose up -d
# Launch S3
./launchs3driver.sh
export VM_TYPE=none
export PUBLIC_IP=10.192.0.3
export DOCKER_REPO_USER=<registry_user>
export DOCKER_REPO_PASS=<registry_password>
export DOCKER_REPO=<registry_url>
# Launch FfDL
cd $GOPATH/src/github.com/IBM/FfDL
make build docker-build docker-push create-volumes deploy
# Run training job
cd etc/examples/
DLAAS_URL=http://10.192.0.3:$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}') DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux train tf-model/manifest.yml tf-model
```

where PUBLIC_IP belongs to the node the REST API runs on. It can be found via `kubectl describe pod ffdl-restapi-...` (look at NodeIP value).

### Build and Deploy

Build FfDL source: `make build`

Build FfDL Docker containers: `make docker-build`

Deploy FfDL to Kubernetes: `make deploy`

[To stop: `make undeploy`]

### Automatic Training
Push data to local S3 with
`make test-push-data`

Run with
`make test-submit`

### Manual Training
The following steps assume you have uploaded the training data to S3, e.g. via `make test-push-data`

Start and follow a training job:

`DLAAS_URL=http://10.192.0.3:30132 DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux train manifest.yml .`

`DLAAS_URL=http://10.192.0.3:30132 DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux logs --follow training-...`

View S3 content in localstack:
`AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws s3 ls --endpoint-url http://10.192.0.4:32083 s3://tf_training_data`

Note that these commands use two distinct URLs for the FfDL REST API as well as S3. They might differ in your case and can be obtained as follows.

The former (i.e. FfDL REST API) uses
`$PUBLIC_IP:$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')`

whereas the latter (i.e. localstack S3 instance) uses
`$(kubectl get po/storage-0 -o=jsonpath='{.status.hostIP}'):$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')`

Alternatively, we could also extract the S3 IP via `sed`:
```
IP=`echo "$(kubectl describe po/storage-0)" | sed -r -n -e '/^Node:/p' | sed 's/^Node:.*\/\(.*\)/\1/'`
```

## Setup of DIND-based non-GPU Kubernetes cluster
The following steps are based on a SoftLayer Ubuntu 16.04 instance.

A fresh SoftLayer instance only contains a root user. Thus, we will create a sudo user as a first step.
```
apt update && apt upgrade
adduser <username>
usermod -aG sudo <username>
```
Exit and login as new user


### Install Docker

```
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
sudo apt-get update && sudo apt-get install -y docker-ce
sudo systemctl status docker
sudo usermod -aG docker ${USER}
```

### Install Docker Compose

```
sudo curl -L https://github.com/docker/compose/releases/download/1.18.0/docker-compose-`uname -s`-`uname -m` -o /usr/local/bin/docker-compose
sudo reboot
```

### Install DIND Kubernetes

```
wget https://cdn.rawgit.com/Mirantis/kubeadm-dind-cluster/master/fixed/dind-cluster-v1.9.sh
chmod +x dind-cluster-v1.9.sh
./dind-cluster-v1.9.sh up
export PATH="$HOME/.kubeadm-dind-cluster:$PATH"
```

### Install Bluemix CLI

Install Bluemix CLI
If on SoftLayer, add `170.225.15.112 public.dhe.ibm.com` to `/etc/hosts`
```
curl -fsSL https://clis.ng.bluemix.net/install/linux | sh
bx plugin install container-registry -r Bluemix
bx login -a https://api.ng.bluemix.net -sso
bx cr login
docker pull registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-plugin:122
docker pull registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-driver:218

sudo apt install libfuse2 libxml2
```

### Install Go(lang)

Find most recent version at https://github.com/golang/go/releases
```
wget https://dl.google.com/go/go1.10.1.linux-amd64.tar.gz
sudo tar -xvf go1.10.1.linux-amd64.tar.gz
sudo mv go /usr/local

mkdir ~/go
echo "export GOROOT=/usr/local/go" >> ~/.profile
echo "export GOPATH=$HOME/go" >> ~/.profile
echo "export PATH=\$GOPATH/bin:\$GOROOT/bin:\$PATH" >> ~/.profile
source ~/.profile
go version && go env
```

### Install kubectl
```
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
```

### Install Helm

Select most recent helm Linux binary from https://github.com/kubernetes/helm/releases
```
cd ~
wget https://storage.googleapis.com/kubernetes-helm/helm-v2.8.2-linux-amd64.tar.gz
tar -zxvf helm-v2.8.2-linux-amd64.tar.gz
sudo mv linux-amd64/helm /usr/local/bin/
helm init
cd $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-plugin/deploy/helm/
sed -i 's/<REGISTRY_URL>/registry.ng.bluemix.net/g' ibmcloud-object-storage-plugin/values.yaml
```

### Setup S3

#### Setup S3 FLEX Driver

```
cd $GOPATH
mkdir -p src/github.ibm.com/alchemy-containers
mkdir $GOPATH/bin
cd armada-storage-s3fs-plugin-ws/src/github.ibm.com/alchemy-containers/
```

From client (or clone git):
```
scp -r ./Downloads/armada-storage-s3fs-plugin/ <username>@<host_ip>:~/go/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-plugin/
cd armada-storage-s3fs-plugin
sudo apt install build-essential
curl https://glide.sh/get | sh
go get github.com/gin-gonic/gin
make
make provisioner
make driver
```

#### Install S3 Driver via Helm Chart

```
helm install ./ibmcloud-object-storage-plugin-security
kubectl get sa -n kube-system | grep object-storage && kubectl get clusterRole | grep object-storage && kubectl get clusterRoleBinding | grep object-storage
helm install ./ibmcloud-object-storage-plugin
kubectl get storageclass | grep s3
kubectl get pods -n kube-system | grep object-storage
kubectl logs po/ibmcloud-object-storage-driver-2j7dx -n kube-system
```

#### Start DIND Kubernetes Cluster and Deploy S3 FLEX Driver

```
./dind-cluster-v1.9.sh up
helm init
docker ps
docker cp ibmcloud-object-storage-driver_218.tar kube-node-1:/
docker cp ibmcloud-object-storage-driver_218.tar kube-node-2:/
docker cp ibmcloud-object-storage-driver_218.tar kube-master:/
docker cp ibmcloud-object-storage-plugin.tar kube-node-1:/
docker cp ibmcloud-object-storage-plugin.tar kube-node-2:/
docker cp ibmcloud-object-storage-plugin.tar kube-master:/
docker exec -it kube-node-1 bash
docker load --input ibmcloud-object-storage-driver_218.tar
docker load --input ibmcloud-object-storage-plugin.tar
apt install -y libfuse2 libxml2
exit
docker exec -it kube-node-2 bash
docker load --input ibmcloud-object-storage-driver_218.tar
docker load --input ibmcloud-object-storage-plugin.tar
apt install -y libfuse2 libxml2
exit docker exec -it kube-master bash
docker load --input ibmcloud-object-storage-driver_218.tar
docker load --input ibmcloud-object-storage-plugin.tar
apt install -y libfuse2 libxml2
exit
```

Create dummy.cm with
```
{
  "account_id": "kube-user",
  "customer_id": "kube-user",
  "cluster_id": "private12345",
  "cluster_name": "private-cluster",
  "cluster_type": "private",
  "datacenter": "local"
}
```

```
kubectl create configmap -n  kube-system cluster-info --from-file=cluster-config.json=./dummy.cm
cd ~/go/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-plugin/deploy/helm
helm install ./ibmcloud-object-storage-plugin-security
helm install ./ibmcloud-object-storage-plugin
cd ~
```

Delete storage-plugin to restart it, e.g. `kubectl delete po/ibmcloud-object-storage-plugin-8cfb4457d-n6tv4 -n kube-system`
```
kubectl delete $(kubectl get pods -n kube-system -o name | grep ibmcloud-object-storage-plugin-) -n kube-system

kubectl get storageclass | grep s3
kubectl get pods -n kube-system | grep object-storage
```

Create secret.yaml with (get keys in base64 with echo -n “<key>” | base64 ):
```
apiVersion: v1
kind: Secret
type: ibm/ibmc-s3fs
metadata:
  name: test-secret
  namespace: default
data:
  access-key: <MY_ACCESS_KEY>
  secret-key: <MY_SECRET_KEY>
```

`kubectl apply -f secret.yaml`

Create pvc.yaml with:
```
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: s3fs-test-pvc-2
  namespace: default
  annotations:
    volume.beta.kubernetes.io/storage-class: "ibmc-s3fs-standard"
    ibm.io/auto-create-bucket: "false"
    ibm.io/auto-delete-bucket: "false"
    ibm.io/bucket: "<YOUR_BUCKET_NAME>"
    ibm.io/endpoint: "https://s3-api.us-geo.objectstorage.softlayer.net"
    ibm.io/region: "us-standard"
    ibm.io/secret-name: "test-secret"
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 8Gi
```

```
kubectl apply -f pvc.yaml
kubectl get pvc
```

Create pod.yaml with:
```
apiVersion: v1
kind: Pod
metadata:
  name: s3fs-test-pod
  namespace: default
spec:
  containers:
  - name: s3fs-test-container
    image: anaudiyal/infinite-loop
    volumeMounts:
    - mountPath: "/mnt/s3fs"
      name: s3fs-test-volume
  volumes:
  - name: s3fs-test-volume
    persistentVolumeClaim:
      claimName: s3fs-test-pvc
```

Run test pod, exec into it and make sure you can see your files.
```bash
kubectl apply -f pod.yaml
kubectl exec -it s3fs-test-pod bash
df -Th | grep s3
ls /mnt/s3fs/
```

In order to use FfDL you can now follow the kick start section above.

### Setup local Docker Registry
```bash
#!/bin/bash

sudo apt-get -y install apache2-utils

mkdir ~/docker-registry && cd $_
mkdir data
mkdir nginx

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
    - 5043:443
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
  # ssl on;
  # ssl_certificate /etc/nginx/conf.d/domain.crt;
  # ssl_certificate_key /etc/nginx/conf.d/domain.key;

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
    # auth_basic "registry.localhost";
    # auth_basic_user_file /etc/nginx/conf.d/registry.password;
    # add_header 'Docker-Distribution-Api-Version' 'registry/2.0' always;

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
htpasswd -c registry.password <user>

openssl genrsa -out devdockerCA.key 2048
openssl req -x509 -new -nodes -key devdockerCA.key -days 10000 -out devdockerCA.crt

openssl genrsa -out domain.key 2048
# NOTE: Enter your domain or IP as common name, e.g. diae3.eecs.umich.edu or 141.213.3.132 !!!
openssl req -new -key domain.key -out dev-docker-registry.com.csr
# Self-sign certificate
openssl x509 -req -in dev-docker-registry.com.csr -CA devdockerCA.crt -CAkey devdockerCA.key -CAcreateserial -out domain.crt -days 10000

sudo mkdir /usr/local/share/ca-certificates/docker-dev-cert
sudo cp devdockerCA.crt /usr/local/share/ca-certificates/docker-dev-cert
sudo update-ca-certificates
```

### Setup NFS
```
sudo apt update
sudo apt install nfs-kernel-server
sudo mkdir -p /nfs/var/nfs/general
sudo chown nobody:nogroup /var/nfs/general
```
`sudo vim /etc/exports` and add
```
/nfs/var/nfs/general    *(rw,sync,no_root_squash,no_subtree_check)
```
Then run `sudo systemctl restart nfs-kernel-server`.

Note that the package `nfs-common` needs to be available on every Kubernetes node. If it is not, you need to exec
into it and install it.

### Clone FfDL
One all prerequisites are installed, please clone FfDL to `$GOPATH/src/github.com/IBM/FfDL/`
and then proceed with the kickstart instructions.

## References

### FfDL
- [FfDL Repository on Github](https://github.com/IBM/FfDL)
- [FfDL IBM Code Open Project Page](https://developer.ibm.com/code/open/projects/fabric-for-deep-learning-ffdl/)
- [developerWorks: Fabric for Deep Learning](https://developer.ibm.com/code/2018/03/20/fabric-for-deep-learning/)
- [developerWorks: Democratize AI with Fabric for Deep Learning (FfDL)](https://developer.ibm.com/code/2018/03/20/democratize-ai-with-fabric-for-deep-learning/)
- [Introducing Fabric for Deep Learning (FfDL) on medium.com](https://medium.com/ibm-watson/introducing-fabric-for-deep-learning-ffdl-542522774775)

### Watson Machine Learning and Watson Studio
- [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning)
- [Deep Learning as a Service with Watson Studio](https://www.ibm.com/cloud/deep-learning)
- [IBM Watson Studio](https://www.ibm.com/cloud/watson-studio)
- [IBM Watson Studio Workshop](https://github.com/PubChimps/DLaaSWorkshop)

### Related IBM Projects
- [Adversarial Robustness Toolbox](https://www.ibm.com/blogs/research/2018/04/ai-adversarial-robustness-toolbox/)
- [Seq2Seq-Vis](http://seq2seq-vis.io/)
- [LSTMVis](https://arxiv.org/abs/1606.07461)

### Kubernetes
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kubernetes DIND Cluster](https://github.com/Mirantis/kubeadm-dind-cluster)
- [developerWorks: Setup GPU-Enabled Kubernetes](https://developer.ibm.com/code/howtos/k8s-kubeadm-gpu-setup)
- [Nvidia Device Plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin)
- [IBM Cloud Container Service](https://www.ibm.com/cloud/container-service)

### localstack, IBM Cloud Object Storage and AWS S3 CLI
- [localstack - a fully functional AWS cloud stack](https://github.com/localstack/localstack)
- [IBM Cloud Object Storage](https://www.ibm.com/cloud/object-storage)
- [AWS S3 CLI Documentation](https://docs.aws.amazon.com/cli/latest/userguide/using-s3-commands.html)

### Papers
- [IBM Deep Learning Service](https://arxiv.org/abs/1709.05871)
- [Scalable Multi-Framework Multi-Tenant Lifecycle Management of Deep Learning Training Jobs](http://learningsys.org/nips17/assets/papers/paper_29.pdf)

### Videos
- [Introduction to Fabric for Deep Learning (FfDL)](https://www.youtube.com/watch?v=aKOqFL7VWhI)
- [Fabric for Deep Learning](https://www.youtube.com/watch?v=nQsYWmkfLP4)

### Slides
- [Fabric for Deep Learning](https://www.slideshare.net/AnimeshSingh/fabric-for-deep-learning-94941117)

## Addendum

### Relevant S3 Commands

Empty COS bucket

`aws s3 rm --recursive --endpoint-url https://s3-api.us-geo.objectstorage.softlayer.net s3://<bucket_name>`

List COS bucket content

`aws s3 ls --endpoint-url https://s3-api.us-geo.objectstorage.softlayer.net s3://<bucket_name>`

Copy file to COS (also works to download COS files if parameters are flipped)

`aws s3 cp --endpoint-url https://s3-api.dal-us-geo.objectstorage.service.networklayer.com <local_file> s3://<target_bucket>`

Upload all files from current directory to COS

`aws s3 cp --endpoint-url https://s3-api.dal-us-geo.objectstorage.service.networklayer.com --recursive . s3://<target_bucket>`

### Installation of Nvidia Docker
Nvidia Docker can be installed as follows (assuming Nvidia drivers are present):
```
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | \
  sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu16.04/amd64/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo pkill -SIGHUP dockerd
docker run --runtime=nvidia --rm nvidia/cuda nvidia-smi
```

### Other useful commands
Delete old training artifacts:
`kubectl delete pod,pvc,pv,sc,deploy,svc,statefulset,secrets --selector training_id`

Get additional logs for debugging:
`kubectl logs po/kube-controller-manager-kube-master -n kube-system`
