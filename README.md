[![build status](https://travis-ci.org/IBM/FfDL.svg?branch=master)](https://travis-ci.org/IBM/FfDL)

# FfDL Core Services

This repository contains the core services of the *FfDL* (Fabric for Deep Learning) platform. FfDL is an operating system "fabric" for Deep Learning

FfDL is a collaboration platform for:
- Framework-independent training of Deep Learning models on distributed hardware
- Open Deep Learning APIs  
- Common instrumentation
- Inferencing in the cloud
- Running Deep Learning hosting in user's private or public cloud

**Note:** This repository is currently work in progress, it is only used for internal testing at the moment.

## Prerequisites

* `kubectl`: the Kubernetes command line interface (https://kubernetes.io/docs/tasks/tools/install-kubectl/)

* `helm`: the Kubernetes package manager (https://helm.sh)

* `docker`: the Docker command-line interface (https://www.docker.com/)

* An existing Kubernetes cluster (e.g., [Minikube](https://github.com/kubernetes/minikube) for local testing).
  Once installed, use the command `make minikube` to start Minikube and set up local network routes. Alternatively,
  use the Vagrant based setup to automatically install a local Kubernetes cluster.

## Steps
1. [Quick Start](#1-quick-start)
  - 1.1 [Installation using Vagrant](#11-installation-using-vagrant)
  - 1.2 [Installation using Minikube](#12-installation-using-minikube)
  - 1.3 [Installation using Kubernetes Cluster](#13-installation-using-kubernetes-cluster)
  - 1.4 [Installation using IBM Cloud Kubernetes Cluster](#14-installation-using-ibm-cloud-kubernetes-cluster)
2. [Test](#2-test)
3. [Monitoring](#3-monitoring)
4. [Development](#4-development)
5. [Detailed Installation Instructions](#5-detailed-installation-instructions)
6. [Detailed Testing Instructions](#6-detailed-testing-instructions)
  - 6.1 [Using FfDL Local Mock S3 Storage](#61-using-ffdl-local-mock-s3-storage)
  - 6.2 [Using Cloud Object Storage](#62-using-cloud-object-storage)
7. [Clean Up](#7-clean-up)
8. [Troubleshooting](#8-troubleshooting)

## 1. Quick Start

There are multiple installation paths for installing FfDL locally ("1-click-install") or
into an existing Kubernetes cluster.

### 1.1 Installation using Vagrant

This is the simplest and recommended option for local testing. The following commands will automatically
spin up a Vagrant box with Kubernetes and the FfDL platform deployed on top of it:
```
export VM_TYPE=vagrant
vagrant up
make deploy
```

### 1.2 Installation using Minikube

If you have Minikube installed on your machine, use these commands to deploy the FfDL platform:
```
export VM_TYPE=minikube
make minikube
make deploy
```

### 1.3 Installation using Kubernetes Cluster

To install FfDL to a proper Kubernetes cluster, make sure `kubectl` points to the right namespace,
then deploy the platform services:
```
export VM_TYPE=none
make deploy
```

### 1.4 Installation using IBM Cloud Kubernetes Cluster

To install FfDL to a proper IBM Cloud Kubernetes cluster, make sure `kubectl` points to the right namespace
and your machine is logged in with `bx login`, then deploy the platform services:
``` shell
export VM_TYPE=ibmcloud
export CLUSTER_NAME=yourClusterName # Replace yourClusterName with your IBM Cloud Cluster Name
make deploy
```

## 2. Test

To submit a simple example training job that is included in this repo (see `etc/examples` folder):

```
make test-submit
```

## 3. Monitoring

The platform ships with a simple Grafana monitoring dashboard. The URL is printed out when running the `deploy` make target.

## 4. Development

Use the following instructions if you want to run a full development build, compile the code, and build the
Docker images locally. 

Install:

* `go`: a working [Go](https://golang.org/) environment is required to build the code

* `glide`: the glide package manager for Go (https://glide.sh)

* `npm`: the Node.js package manager (https://www.npmjs.com) for building the Web UI

* `Go` is very specific about directory layouts. Make sure to set your `$GOPATH` and clone this repo to a directory
`$GOPATH/src/github.com/IBM/FfDL` before proceeding with the next steps.


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

## 5. Detailed Installation Instructions

1. First, clone this repository and install the helm tiller on your Kubernetes cluster.
``` shell
helm init

# Make sure the tiller pod is Running before proceeding to the next step.
kubectl get pods --all-namespaces | grep tiller-deploy
# kube-system   tiller-deploy-fb8d7b69c-pcvc2              1/1       Running
```

2. Now let's install all the necessary FfDL components using helm install.
``` shell
helm install .
```
> Note: If you want to upgrade an older version of FfDL, run
> `helm upgrade $(helm list | grep ffdl | awk '{print $1}' | head -n 1) .`

Make sure all the FfDL components are installed and running before moving to the next step.
``` shell
kubectl get pods
# NAME                                 READY     STATUS    RESTARTS   AGE
# alertmanager-7cf6b988b9-h9q6q        1/1       Running   0          5h
# etcd0                                1/1       Running   0          5h
# ffdl-lcm-65bc97bcfd-qqkfc            1/1       Running   0          5h
# ffdl-restapi-8777444f6-7jfcf         1/1       Running   0          5h
# ffdl-trainer-768d7d6b9-4k8ql         1/1       Running   0          5h
# ffdl-trainingdata-866c8f48f5-ng27z   1/1       Running   0          5h
# ffdl-ui-5bf86cc7f5-zsqv5             1/1       Running   0          5h
# mongo-0                              1/1       Running   0          5h
# prometheus-5f85fd7695-6dpt8          2/2       Running   0          5h
# pushgateway-7dd8f7c86d-gzr2g         2/2       Running   0          5h
# storage-0                            1/1       Running   0          5h

helm status $(helm list | grep ffdl | awk '{print $1}' | head -n 1) | grep STATUS:
# STATUS: DEPLOYED
```

3. Run the following script to configure Grafana for monitoring FfDL using the logging information from prometheus.
> Note: If you are using a IBM Cloud Cluster, make sure you are logged in with `bx login`.

``` shell
# If your Cluster is running on Vagrant or Minikube, replace "ibmcloud" to "vagrant" | "minikube"
export VM_TYPE=ibmcloud
# Replace yourClusterName with your IBM Cloud Cluster Name if your cluster is on IBM Cloud.
export CLUSTER_NAME=yourClusterName
./bin/grafana.init.sh
```

4. Lastly, run the following commands to obtain your Grafana, FfDL Web UI, and FfDL restapi endpoints.
``` shell
# Note: $(make --no-print-directory kubernetes-ip) simply gets the Public IP for your cluster.
node_ip=$(make --no-print-directory kubernetes-ip)

# Obtain all the necessary NodePorts for Grafana, Web UI, and RestAPI.
grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
ui_port=$(kubectl get service ffdl-ui -o jsonpath='{.spec.ports[0].nodePort}')
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')

# Echo statements to print out Grafana and Web UI URLs.
echo "Monitoring dashboard: http://$node_ip:$grafana_port/ (login: admin/admin)"
echo "Web UI: http://$node_ip:$ui_port/#/login?endpoint=$node_ip:$restapi_port&username=test-user"
```

Congratulation, FfDL is now running on your Cluster.

## 6. Detailed Testing Instructions

In this example, we will run some simple jobs to train a convolutional network model using TensorFlow and Caffe. We will download a set of
MNIST handwritten digit images, store them with Object Storage, and use the FfDL CLI to train a handwritten digit classification model.

> Note: For Minikube, make sure you have the latest TensorFlow Docker image by running `docker pull tensorflow/tensorflow`

### 6.1. Using FfDL Local Mock S3 Storage

1. Run the following commands to obtain the Mock S3 storage endpoint from your cluster.
```shell
node_ip=$(make --no-print-directory kubernetes-ip)
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$node_ip:$s3_port
```

2. Next, set up the default Mock S3 storage access ID and KEY. Then create S3 buckets for all the necessary training data and models.
```shell
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;

s3cmd="aws --endpoint-url=$s3_url s3"
$s3cmd mb s3://tf_training_data
$s3cmd mb s3://tf_trained_model
$s3cmd mb s3://mnist_lmdb_data
$s3cmd mb s3://dlaas-trained-models
```

3. Now, create a temporary repository, download the necessary images for training and labeling our TensorFlow model, and upload those images
to your S3 tf_training_data bucket.

```shell
mkdir tmp
for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz;
do
  test -e tmp/$file || wget -q -O tmp/$file http://yann.lecun.com/exdb/mnist/$file
  $s3cmd cp tmp/$file s3://tf_training_data/$file
done
```

4. Next, let's download all the necessary training and testing images in LMDB format for our Caffe model
and upload those images to your S3 mnist_lmdb_data bucket.

```shell
for phase in train test;
do
  for file in data.mdb lock.mdb;
  do
    tmpfile=tmp/$phase.$file
    test -e $tmpfile || wget -q -O $tmpfile https://github.com/albarji/caffe-demos/raw/master/mnist/mnist_"$phase"_lmdb/$file
    $s3cmd cp $tmpfile s3://mnist_lmdb_data/$phase/$file
  done
done
```

5. Now you should have all the necessary training data set in your Mock S3 storage. Let's go ahead to set up your restapi endpoint
and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI(executable
binary).

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default TensorFlow model
CLI_CMD=cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

Congratulation, you had submitted your first job on FfDL. You can check your FfDL status either from the FfDL UI or simply run `$CLI_CMD list`

6. Since it's simple and straightforward to submit jobs with different deep learning framework on FfDL, let's try to run a Caffe Job.

```shell
$CLI_CMD train etc/examples/caffe-model/manifest.yml etc/examples/caffe-model
```

Congratulation, now you know how to deploy jobs with different deep learning framework. To learn more about your job execution results,
you can simply run `$CLI_CMD logs <MODEL_ID>`

> If you no longer need any of the MNIST dataset we used in this example, you can simply delete the tmp repository.

7. (Experimental) After you done with step 4, if you want to run your job via the FfDL UI, simply upload
`tf-model.zip` and `manifest.yml` (The default TensorFlow model) in the `etc/examples/` repository as shown below.
Then, click `Submit Training Job` to run your job.

![ui-example](docs/images/ui-example.png)

### 6.2. Using Cloud Object Storage

Due to the cost of having multiple Object Storage buckets, we will only demonstrate how to run a TensorFlow Job using Cloud Object Storage.

> Note: This also can be done with other Cloud providers' Object Storage, but we will demonstrate how to use IBM Cloud Object Storage in this instructions.

1. Provision an S3 based Object Storage from your Cloud provider. Take note of your Authentication Endpoints, Access Key ID and Secret.

> For IBM Cloud, you can provision an Object Storage from [IBM Cloud Dashboard](https://console.bluemix.net/catalog/infrastructure/cloud-object-storage?taxonomyNavigation=apps) or from [SoftLayer Portal](https://control.softlayer.com/storage/objectstorage).

2. Setup your S3 command with the Object Storage credentials you just obtained.

```shell
s3_url=http://<Your object storage Authentication Endpoints>
export AWS_ACCESS_KEY_ID=<Your object storage Access Key ID>
export AWS_SECRET_ACCESS_KEY=<Your object storage Access Key Secret>

s3cmd="aws --endpoint-url=$s3_url s3"
```

3. Next, let create 2 buckets, one for storing the training data and another one for storing the training result.
```shell
trainingDataBucket=<unique bucket name for training data storage>
trainingResultBucket=<unique bucket name for training result storage>

$s3cmd mb s3://$trainingDataBucket
$s3cmd mb s3://$trainingResultBucket
```

4. Now, create a temporary repository, download the necessary images for training and labeling our TensorFlow model, and upload those images to your training data bucket.

```shell
mkdir tmp
for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz;
do
  test -e tmp/$file || wget -q -O tmp/$file http://yann.lecun.com/exdb/mnist/$file
  $s3cmd cp tmp/$file s3://$trainingDataBucket/$file
done
```

5. Next, we need to modify our example job to use your Cloud Object Storage using the following sed commands.
```shell
if [ "$(uname)" = "Darwin" ]; then
  sed -i '' s#"tf_training_data"#"$trainingDataBucket"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"tf_trained_model"#"$trainingResultBucket"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"http://s3.default.svc.cluster.local"#"$s3_url"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"user_name: test"#"user_name: $AWS_ACCESS_KEY_ID"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"password: test"#"password: $AWS_SECRET_ACCESS_KEY"# etc/examples/tf-model/manifest.yml
else
  sed -i s#"tf_training_data"#"$trainingDataBucket"# etc/examples/tf-model/manifest.yml
  sed -i s#"tf_trained_model"#"$trainingResultBucket"# etc/examples/tf-model/manifest.yml
  sed -i s#"http://s3.default.svc.cluster.local"#"$s3_url"# etc/examples/tf-model/manifest.yml
  sed -i s#"user_name: test"#"user_name: $AWS_ACCESS_KEY_ID"# etc/examples/tf-model/manifest.yml
  sed -i s#"password: test"#"password: $AWS_SECRET_ACCESS_KEY"# etc/examples/tf-model/manifest.yml
fi
```

6. Now you should have all the necessary training data set in your training data bucket. Let's go ahead to set up your restapi endpoint
and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI(executable
binary).

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default TensorFlow model
CLI_CMD=cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

## 7. Clean Up
If you want to remove FfDL from your cluster, simply use the command below or run `helm delete <your FfDL release name>`
```shell
helm delete $(helm list | grep ffdl | awk '{print $1}' | head -n 1)
```

## 8. Troubleshooting

* FfDL has only been tested under Mac OS and Linux 

* The default Minikube driver under Mac OS is VirtualBox, which is known for having issues with networking.
  We generally recommend Mac OS users to install Minikube using the xhyve driver.
  
* Also, when testing locally with Minikube, make sure to point the `docker` CLI to Minikube's Docker daemon:

   ```
   eval $(minikube docker-env)
   ```
* If you run into DNS name resolution issues using Minikube, make sure that the system uses only `10.0.0.10`
  as the single nameserver. Using multiple nameservers can result in problems, in particular under Mac OS.

* If `glide install` fails with an error complaining about non-existing paths (e.g., "Without src, cannot continue"),
  make sure to follow the standard Go directory layout (see [Prerequisites section]{#Prerequisites}).

* To remove FfDL on your Cluster, simply run `make undeploy`
