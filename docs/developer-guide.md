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
make deploy-plugin # Deploy S3 storage plugin
make deploy # Deploy FfDL
```

## Troubleshooting
If your Object Storage Driver is not successfully installed on your Kubernetes, you can following the step by step instructions at [ibmcloud-object-storage-plugin](https://github.com/IBM/ibmcloud-object-storage-plugin).

If you encounter other issues, please take a look at the experimental [troubleshooting.md](./troubleshooting.md).


## Scripts
There are experimental scripts to setup FfDL including all of its dependencies like Docker, Go, Kubernetes, S3 drivers and the Docker registry starting from an empty SoftLayer VM.
You can find them in `bin/dind_scripts`. To start on a fresh VM login as root and run:
```bash
apt install -y git software-properties-common
mkdir -p /home/ffdlr/go/src/github.com/IBM/ && cd $_ && git clone https://github.com/fplk/FfDL.git && cd FfDL && git checkout merge_20180514_1536
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

## Enable device plugin for GPU workloads with development build

Please modify the `resourceGPU` under [lcm/service/lcm/container_helper.go](../lcm/service/lcm/container_helper.go#L530) and [lcm/service/lcm/resources.go](../lcm/service/lcm/resources.go#L149) to `"nvidia.com/gpu"` and rebuild the lcm image to enable device plugin for all GPU workloads on your development build.

## Enable custom learner images with development build

Please uncomment the following section under [trainer/trainer/frameworks.go](../trainer/trainer/frameworks.go#L39) and rebuild the trainer image to enable custom learner images from any users. Alternatively, you can use the pre-built images `ffdl/ffdl-trainer:customizable` on DockerHub.

``` go
if fwName == "custom" {
  return true, ""
}
```

After you deployed `ffdl-trainer` with custom image feature, you can use your custom learner images by changing the `framework.name` to **custom** and `framework.version` to your learner image in your training job's manifest.yml file. If you are using any private registry, you need to enable access to the private registry in your Kubernetes default namespace.
