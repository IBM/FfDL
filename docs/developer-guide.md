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

Define the folloing environment variables:
```shell
export SHARED_VOLUME_STORAGE_CLASS=<StorageClass>
export PUBLIC_IP=<IP_TO_CLUSTER>
export DOCKER_REPO_USER=<REPOSITORY_USER>
export DOCKER_PULL_POLICY=Always
export DOCKER_REPO_PASS=<PASSWORD_TO_YOUR_REPOSITORY>
export DOCKER_NAMESPACE=<NAMESPACE_ON_IBM_CLOUD>
export DOCKER_REPO=<registry endpoint> # registry.ng.bluemix.net
export VM_TYPE=none
export HAS_STATIC_VOLUMES=True
```

Compile the code, generate certificates, and build the Docker images via:
```shell
make build
make gen-certs
make docker-build
```
If you want to push the images you just built, run:
```shell
make docker push
```

Make sure `kubectl` points to the right target context/namespace, then deploy the services to your Kubernetes
environment (using `helm`):
```shell
make deploy-plugin
make deploy
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
