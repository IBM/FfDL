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

> For Minikube, run `eval $(minikube docker-env)` to point your Docker CLI to the Minikube's Docker daemon.

Then, fetch the dependencies via:
```
glide install
```
Compile the code and build the Docker images via:
```
make build
make docker-build
```

Make sure `kubectl` points to the right target context/namespace, then deploy the services to your Kubernetes
environment (using `helm`):
```
make deploy
```

## Enable accelerator for GPU workloads with development build

To enable accelerator for all GPU workloads on your development build, change [values.yaml](../values.yaml#L30)'s `lcm.GPU_resources` to **accelerator** and redeploy FfDL.

## Enable custom learner images with development build

To enable custom learner images from any users, change [values.yaml](../values.yaml#L17)'s `trainer.customizable` to **true** and redeploy FfDL.

After you deployed `ffdl-trainer` with custom image feature, you can use your custom learner images by changing the `framework.name` to **custom** and `framework.version` to your learner image in your training job's manifest.yml file. If you are using any private registry, you need to enable access to the private registry in your Kubernetes default namespace.

## Customize helper pod specs

If your Kubernetes Cluster has a lot of resources and you want to enhance the training jobs execution speed, you can modify the helper pod spec under [values.yaml](../values.yaml).

* `storeResultsMilliCPU`: Store Results is responsible for storing all the training result. Increasing this number along with `storeResultsMemInMB` can accelerate the storing stage of the training job. Default = 20.
* `storeResultsMemInMB`: See above. Default = 100
* `loadModelMilliCPU` : Load Model is responsible for loading any model data you submitted to the learner container. Increasing this number along with `loadModelMemInMB` can accelerate the download stage of the training job. Default = 20.
* `loadModelMemInMB`: See above. Default = 50.
* `loadTrainingDataMilliCPU`: Load Training Data is responsible for loading any training data in the object storage to the learner container. Increasing this number along with `loadTrainingDataMemInMB` can accelerate the download stage of the training job. Default = 20.
* `loadTrainingDataMemInMB`: See above. Default = 300.
* `logCollectorMilliCPU`: Log Collector is responsible for collecting any log and metadata from the learner containers. Increasing this number along with `logCollectorMemInMB` can allow Log Collector to collect more logs and TF summary data. Default = 60.
* `logCollectorMemInMB`: See above. Default = 300.
* `controllerMilliCPU`: Controller is responsible for giving instructions (e.g. halting jobs) to the learners. Increasing this number along with `controllerMemInMB` can let the training job to be more responsive for your changes. Default = 20.
* `controllerMemInMB`: See above. Default = 100.


## Deploy FfDL in a dedicated namespace

You need to modify the following sections to deploy FfDL in a non default namespace.
1. Change [values.yaml](../values.yaml#L1)'s `namespace` section. (Note that the helm chart will create the namespace for you)
2. Change [values.yaml](../values.yaml#L71)'s `objectstore.auth_url` section with the new KubeDNS link that targets the new namespace. (e.g.  `http://s3.<namespace>.svc.cluster.local`)
3. Since all the scripts and command are targeting the default namespace, you need to change the namespace preference after you run helm install.
 ```shell
 kubectl config set-context $(kubectl config current-context) --namespace=<namespace>
 ```
4. If you are using the local S3 for storing your dataset, make sure you changed the KubeDNS auth_url in your training job's manifest file to targets the new namespace. (e.g.  `http://s3.<namespace>.svc.cluster.local`)
