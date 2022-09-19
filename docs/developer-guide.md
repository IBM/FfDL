# Developer Guide

## Run a full development build

Use the following instructions if you want to run a full development build, compile the code, and build the
Docker images locally.

Install:

* `go`: a working [Go](https://golang.org/) environment is required to build the code

* `glide`: the glide package manager for Go (https://glide.sh)

* `npm`: the Node.js package manager (https://www.npmjs.com) for building the Web UI

* `Go` is very specific about directory layouts. Make sure to set your `$GOPATH` and clone this repo to a directory
`$GOPATH/src/github.com/IBM/FfDL` before proceeding with the next steps.

Then, fetch the dependencies via:
```shell
glide install
```

Define the following environment variables:
```shell
export SHARED_VOLUME_STORAGE_CLASS=<StorageClass> # "" for DIND, "ibmc-file-gold" for IBM Cloud
export PUBLIC_IP=<IP_TO_CLUSTER> # One exposed IP of cluster
export DOCKER_REPO=<registry endpoint> # Registry if used, e.g. for DockerHub is "docker.io"
export DOCKER_REPO_USER=<REPOSITORY_USER> # Container Registry Username
export DOCKER_REPO_PASS=<PASSWORD_TO_YOUR_REPOSITORY> # Container Registry Password
export DOCKER_NAMESPACE=<NAMESPACE_ON_IBM_CLOUD> # Container Registry Namespace
export DOCKER_PULL_POLICY=Always # Keep IfNotPresent if not pushing to registry, e.g. for Minikube
export VM_TYPE=none
export HAS_STATIC_VOLUMES=True
export IMAGE_TAG=user-$(whoami)
export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before proceeding to the next step
```

Compile the code, generate certificates, and build the Docker images via:
```shell
make build             # Compile FfDL
make gen-certs         # Generate certificated
make docker-build-base # Build base Docker images
make docker-build      # Build Docker images
```

If you want to push the images you just built, run:
```shell
make docker-push # Push built Docker images to registry, not used for Minikube
```

Make sure `kubectl` points to the right target context/namespace, then deploy the services to your Kubernetes
environment (using `helm`):
```shell
kubectl config set-context $(kubectl config current-context) --namespace=$NAMESPACE # Set your current-context to the FfDL namespace

# Configure s3 driver on the cluster
helm install docs/helm-charts/ibmcloud-object-storage-plugin --name ibmcloud-object-storage-plugin --set namespace=$NAMESPACE
# Deploy all the helper micro-services for ffdl
helm install docs/helm-charts/ffdl-helper --name ffdl-helper \
--set namespace=$NAMESPACE \
--set shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS \
--set localstorage=false \ # set to true if your cluster doesn't have any storage class
--set prometheus.deploy=true \ # set to false if you don't need prometheus logging for ffdl
--wait
# Deploy all the core ffdl services.
helm install docs/helm-charts/ffdl-core --name ffdl-core \
--set namespace=$NAMESPACE \
--set lcm.shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS \
--set docker.registry=$DOCKER_REPO \
--set docker.namespace=$DOCKER_NAMESPACE \
--set docker.pullPolicy=$DOCKER_PULL_POLICY \
--set trainer.version=${IMAGE_TAG},restapi.version=${IMAGE_TAG},lcm.version=${IMAGE_TAG},trainingdata.version=${IMAGE_TAG},databroker.tag=${IMAGE_TAG},databroker.version=${IMAGE_TAG},webui.version=${IMAGE_TAG} \
--wait
```

## Troubleshooting
If your Object Storage Driver is not successfully installed on your Kubernetes, you can following the step by step instructions at [ibmcloud-object-storage-plugin](https://github.com/IBM/ibmcloud-object-storage-plugin).

If you encounter other issues, please take a look at the experimental [troubleshooting.md](./troubleshooting.md).


## Scripts
There are experimental scripts to setup FfDL including all of its dependencies like Docker, Go, Kubernetes, S3 drivers and the Docker registry starting from an empty SoftLayer VM.
You can find them in `bin/dind_scripts`. To start on a fresh VM login as root and run:
```bash
apt install -y git software-properties-common
mkdir -p /home/ffdlr/go/src/github.com/IBM/ && cd $_ && git clone https://github.com/IBM/FfDL.git && cd FfDL
cd bin/dind_scripts/
chmod +x create_user.sh
. create_user.sh
```
Then log back into the VM as the user `ffdlr` and run
```bash
cd /home/ffdlr/go/src/github.com/IBM/FfDL/bin/dind_scripts/
sudo chmod +x experimental_master.sh
. experimental_master.sh
```

## Instructions on GPU workloads

Please refer to the [gpu-guide.md](gpu-guide.md) for more details.

## Enable custom learner images with development build

Custom learner is enabled by default. To use it, simply put `custom` as your framework name and set your custom learner image path as the framework version.
