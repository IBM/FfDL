*Read this in other languages: [中文](README-cn.md).*

[![build status](https://travis-ci.org/IBM/FfDL.svg?branch=master)](https://travis-ci.org/IBM/FfDL)

# Fabric for Deep Learning (FfDL)

### [Latest: PyTorch 1.0 and ONNX support now in FfDL](/etc/examples/PyTorch.md)

This repository contains the core services of the *FfDL* (Fabric for Deep Learning) platform. FfDL is an operating system "fabric" for Deep Learning. It is a collaboration platform for:
- Framework-independent training of Deep Learning models on distributed hardware
- Open Deep Learning APIs  
- Running Deep Learning hosting in user's private or public cloud

![ffdl-architecture](docs/images/ffdl-architecture.png)

To know more about the architectural details, please read the [design document](design/design_docs.md). If you are looking for demos, slides, collaterals, blogs, webinars and other materials related to FfDL, please [find them here](demos)

## Prerequisites

* `kubectl`: The Kubernetes command line interface (https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* `helm`: The Kubernetes package manager (https://helm.sh)
* `docker`: The Docker command-line interface (https://www.docker.com/)
* `S3 CLI`: The [command-line interface](https://aws.amazon.com/cli/) to configure your Object Storage
* An existing Kubernetes cluster (e.g., [Kubeadm-DIND](https://github.com/kubernetes-sigs/kubeadm-dind-cluster#using-preconfigured-scripts) for local testing).
  <!-- For Minikube, use the command `make minikube` to start Minikube and set up local network routes. Minikube **v0.25.1** is tested with Travis CI. -->
* Follow the appropriate instructions for standing up your Kubernetes cluster using [IBM Cloud Public](https://github.com/IBM/container-journey-template/blob/master/README.md) or [IBM Cloud Private](https://github.com/IBM/deploy-ibm-cloud-private/blob/master/README.md)
* The minimum capacity requirement for FfDL is 4GB Memory and 3 CPUs.

## Usage Scenarios

* If you are getting started and want to setup your own FfDL deployment, please follow the steps [below](#1-quick-start).
* If you have a FfDL deployment up and running, you can jump to [FfDL User Guide](docs/user-guide.md) to use FfDL for training your deep learning models.
* If you want to leverage Jupyter notebooks to launch training on your FfDL cluster, please follow [these instructions](etc/notebooks/art)
* If you have FfDL confiugured to use GPUs, and want to train using GPUs, follow steps [here](docs/gpu-guide.md)
* To invoke [Adversarial Robustness Toolbox](https://github.com/IBM/adversarial-robustness-toolbox) to find vulnerabilities in your models, follow the [instructions here](etc/notebooks/art)
* To deploy your trained models, follow [the integration guide with Seldon](community/FfDL-Seldon)
* If you have used FfDL to train your models, and want to use a GPU enabled public cloud hosted service for further training and serving, please follow instructions [here](etc/converter/ffdl-wml.md) to train and serve your models using [Watson Studio Deep Learning](https://www.ibm.com/cloud/deep-learning) service.

## Steps

1. [Quick Start](#1-quick-start)
  - 1.1 [Installation using Kubeadm-DIND](#11-installation-using-kubeadm-dind)
  - 1.2 [Installation using Kubernetes Cluster](#12-installation-using-kubernetes-cluster)
2. [Test](#2-test)
3. [Monitoring](#3-monitoring)
4. [Development](#4-development)
5. [Clean Up](#7-clean-up)
6. [Troubleshooting](#8-troubleshooting)
7. [References](#9-references)

## 1. Quick Start

There are multiple installation paths for installing FfDL into an existing Kubernetes cluster. Below are the steps for quick install. If you want to follow more detailed step by step instructions , please visit [the detailed installation guide](docs/detailed-installation-guide.md)

> If you are using bash shell, you can modify the necessary environment variables in `env.txt` and export all of them using the following commands
>  ```shell
>  source env.txt
>  export $(cut -d= -f1 env.txt)
>  ```

### 1.1 Installation using Kubeadm-DIND

If you have [Kubeadm-DIND](https://github.com/kubernetes-sigs/kubeadm-dind-cluster#using-preconfigured-scripts) installed on your machine, use these commands to deploy the FfDL platform:
``` shell
export VM_TYPE=dind
export PUBLIC_IP=localhost
export SHARED_VOLUME_STORAGE_CLASS="";
export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before running the make commands below

make deploy-plugin
make quickstart-deploy
```

### 1.2 Installation using Kubernetes Cluster

To install FfDL to any proper Kubernetes cluster, make sure `kubectl` points to the right namespace,
then deploy the platform services:
> Note: For PUBLIC_IP, put down one of your Cluster Public IP that can access your Cluster's NodePorts. For IBM Cloud, you can get your Public IP with `bx cs workers <cluster_name>`.

``` shell
export VM_TYPE=none
export PUBLIC_IP=<Cluster Public IP>
export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before running the make commands below

# Change the storage class to what's available on your Cloud Kubernetes Cluster.
export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold";

helm install helm-charts/storage-plugin --set namespace=$NAMESPACE # Configure s3 driver on the cluster
helm install helm-charts/storage-config --set namespace=$NAMESPACE,shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS # Prepare any local/static volume as the shared file system
helm install helm-charts/ffdl-helper --set namespace=$NAMESPACE --wait # Deploy all the helper micro-services for ffdl
helm install helm-charts/ffdl-core --set namespace=$NAMESPACE,lcm.shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS --wait # Deploy all the core ffdl services.
```

## 2. Test

To submit a simple example training job that is included in this repo (see `etc/examples` folder):

``` shell
make test-push-data-s3
make test-job-submit
```

## 3. Monitoring

The platform ships with a simple Grafana monitoring dashboard. The URL is printed out when running the `deploy` make target.

## 4. Development

Please refer to the [developer guide](docs/developer-guide.md) for more details.

## 5. Clean Up
If you want to remove FfDL from your cluster, simply use the following commands.
```shell
helm delete $(helm list | grep ffdl | awk '{print $1}' | head -n 1)
```
If you want to remove the storage driver and pvc from your cluster, run:
```shell
kubectl delete pvc static-volume-1
helm delete $(helm list | grep ibmcloud-object-storage-plugin | awk '{print $1}' | head -n 1)
```
For Kubeadm-DIND, you need to kill your forwarded ports. Note that the below command will kill all the ports that are created with `kubectl`.
```shell
kill $(lsof -i | grep kubectl | awk '{printf $2 " " }')
```

## 6. Troubleshooting

* FfDL has only been tested under Mac OS and Linux
<!-- * The default Minikube driver under Mac OS is VirtualBox, which is known for having issues with networking.
  We generally recommend Mac OS users to install Minikube using the xhyve driver.

* Also, when testing locally with Minikube, make sure to point the `docker` CLI to Minikube's Docker daemon:
   ```
   eval $(minikube docker-env)
   ```
* If you run into DNS name resolution issues using Minikube, make sure that the system uses only `10.0.0.10`
  as the single nameserver. Using multiple nameservers can result in problems, in particular under Mac OS. -->

* If `glide install` fails with an error complaining about non-existing paths (e.g., "Without src, cannot continue"),
  make sure to follow the standard Go directory layout (see [Prerequisites section](#prerequisites)).

* To remove FfDL on your Cluster, simply run `make undeploy`

* When using the FfDL CLI to train a model, make sure your directory path doesn't have slashes `/` at the end.

* If your job is stuck in pending stage, you can try to redeploy the plugin with `helm install storage-plugin --set dind=true,cloud=false` for Kubeadm-DIND and `helm install storage-plugin` for general Kubernetes Cluster. Also, double check your training job manifest file to make sure you have the correct object storage credentials.

## 7. References

Based on IBM Research work in Deep Learning.

* B. Bhattacharjee et al., "IBM Deep Learning Service," in IBM Journal of Research and Development, vol. 61, no. 4, pp. 10:1-10:11, July-Sept. 1 2017.   https://arxiv.org/abs/1709.05871

* Scott Boag, et al. Scalable Multi-Framework Multi-Tenant Lifecycle Management of Deep Learning Training Jobs, In Workshop on ML Systems at NIPS'17, 2017. http://learningsys.org/nips17/assets/papers/paper_29.pdf
