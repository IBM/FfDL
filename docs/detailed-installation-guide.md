
# Detailed Installation Guide


1. [Detailed Installation Instructions](#1-detailed-installation-instructions)
2. [Detailed Testing Instructions](#2-detailed-testing-instructions)
  - 2.1 [Using FfDL Local S3 Based Object Storage](#21-using-ffdl-local-s3-based-object-storage)
  - 2.2 [Using Cloud Object Storage](#22-using-cloud-object-storage)

## 1. Detailed Installation Instructions

1. First, Install the helm tiller on your Kubernetes cluster.
``` shell
helm init
```

2.1. Installation using Kubernetes Cluster

To install FfDL to any proper Kubernetes cluster, make sure `kubectl` points to the right namespace,
then deploy the platform services:

``` shell
export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before running the make commands below
export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold" # Change the storage class to what's available on your Cloud Kubernetes Cluster.

# Configure s3 driver on the cluster
helm install ibm-cloud-storage-plugin --name ibm-cloud-storage-plugin --repo https://ibm.github.io/FfDL/helm-charts --set namespace=$NAMESPACE
# Deploy all the helper micro-services for ffdl
helm install ffdl-helper --name ffdl-helper --repo https://ibm.github.io/FfDL/helm-charts \
  --set namespace=$NAMESPACE \
  --set shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS \
  --set localstorage=false \ # set to true if your cluster doesn't have any storage class
  --set prometheus.deploy=true \ # set to false if you don't need prometheus logging for ffdl
  --wait
# Deploy all the core ffdl services.
helm install ffdl-core --name ffdl-core --repo https://ibm.github.io/FfDL/helm-charts \
  --set namespace=$NAMESPACE \
  --set lcm.shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS \
  --wait
```

2.2 Installation using Kubeadm-DIND

If you don't have a Kubernetes Cluster, you can create a [Kubeadm-DIND](https://github.com/kubernetes-sigs/kubeadm-dind-cluster#using-preconfigured-scripts) Kubernetes Cluster on your local machine. We recommend you give at least 4 CPUs and 8GB of memory to your Docker.
> For Mac users, visit the instructions on the [Docker website](https://docs.docker.com/docker-for-mac/#advanced) and learn how to give more memory to your Docker.

If you have [Kubeadm-DIND](https://github.com/kubernetes-sigs/kubeadm-dind-cluster#using-preconfigured-scripts) installed on your machine, use these commands to deploy the FfDL platform:
``` shell
export SHARED_VOLUME_STORAGE_CLASS=""
export NAMESPACE=default

./bin/s3_driver.sh # Copy the s3 drivers to each of the DIND node
helm install ibm-cloud-storage-plugin --name ibm-cloud-storage-plugin --repo https://ibm.github.io/FfDL/helm-charts --set namespace=$NAMESPACE,cloud=false
helm install ffdl-helper --name ffdl-helper --repo https://ibm.github.io/FfDL/helm-charts --set namespace=$NAMESPACE,shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS,localstorage=true --wait
helm install ffdl-core --name ffdl-core --repo https://ibm.github.io/FfDL/helm-charts --set namespace=$NAMESPACE,lcm.shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS --wait

# Forward the necessary microservices from the DIND cluster to your localhost.
./bin/dind-port-forward.sh
```

Congratulation, FfDL is now running on your Cluster. Now you can go to [Step 2](#2-detailed-testing-instructions) to run some sample jobs or go to the [user guide](docs/user-guide.md) to learn about how to run and deploy your custom models.

## 2. Detailed Testing Instructions

In this example, we will run some simple jobs to train a convolutional network model using TensorFlow. We will download a set of
MNIST handwritten digit images, store them with Object Storage, and use the FfDL CLI to train a handwritten digit classification model.

> Note: For PUBLIC_IP, put down one of your Cluster Public IP that can access your Cluster's NodePorts. You can check your Cluster Public IP with `kubectl get nodes -o wide`.
> For IBM Cloud, you can get your Public IP with `bx cs workers <cluster_name>`.

### 2.1. Using FfDL Local S3 Based Object Storage

1. Clone this repository and run the following commands to obtain the object storage endpoint from your cluster.
```shell
PUBLIC_IP=<Cluster Public IP> # Put down localhost if you are running with Kubeadm-DIND
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$PUBLIC_IP:$s3_port
```

2. Next, set up the default object storage access ID and KEY. Then create buckets for all the necessary training data and models.
```shell
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;

s3cmd="aws --endpoint-url=$s3_url s3"
$s3cmd mb s3://tf_training_data
$s3cmd mb s3://tf_trained_model
```

3. Now, create a temporary repository, download the necessary images for training and labeling our TensorFlow model, and upload those images
to your tf_training_data bucket.

```shell
mkdir tmp
for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz;
do
  test -e tmp/$file || wget -q -O tmp/$file http://yann.lecun.com/exdb/mnist/$file
  $s3cmd cp tmp/$file s3://tf_training_data/$file
done
```

4. Now you should have all the necessary training data set in your object storage. Let's go ahead to set up your restapi endpoint
and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI (executable
binary).

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$PUBLIC_IP:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;
```

Replace the default object storage path with your s3_url. You can skip this step if your already modified the object storage path with your s3_url.
```shell
if [ "$(uname)" = "Darwin" ]; then
  sed -i '' s/s3.default.svc.cluster.local/$PUBLIC_IP:$s3_port/ etc/examples/tf-model/manifest.yml
else
  sed -i s/s3.default.svc.cluster.local/$PUBLIC_IP:$s3_port/ etc/examples/tf-model/manifest.yml
fi
```

Define the FfDL command line interface and run the training job with our default TensorFlow model
```shell
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

Congratulations, you had submitted your first job on FfDL. You can check your FfDL status either from the FfDL UI or simply run `$CLI_CMD list`. To learn more about your job execution results, you can simply run `$CLI_CMD logs <MODEL_ID>`

> You can learn about how to create your own model definition files and `manifest.yaml` at [user guild](docs/user-guide.md#2-create-new-models-with-ffdl).

5. If you want to run your job via the FfDL UI, simply run the below command to create your model zip file.

```shell
# Replace tf-model with the model you want to zip
pushd etc/examples/tf-model && zip ../tf-model.zip * && popd
```

Then, upload `tf-model.zip` and `manifest.yml` (The default TensorFlow model) in the `etc/examples/` repository as shown below.
Then, click `Submit Training Job` to run your job.

![ui-example](images/ui-example.png)

> If you no longer need any of the MNIST dataset we used in this example, you can simply delete the `tmp` repository.

### 2.2. Using Cloud Object Storage

In this section we will demonstrate how to run a TensorFlow job with training data stored in Cloud Object Storage.

1. Provision an S3 based Object Storage from your Cloud provider. Take note of your Authentication Endpoints, Access Key ID and Secret.

> For IBM Cloud, you can provision an Object Storage from [IBM Cloud Dashboard](https://console.bluemix.net/catalog/infrastructure/cloud-object-storage?taxonomyNavigation=apps) or from [SoftLayer Portal](https://control.softlayer.com/storage/objectstorage).

2. Setup your S3 command with the Object Storage credentials you just obtained.

```shell
s3_url=http://<Your object storage Authentication Endpoints>
export AWS_ACCESS_KEY_ID=<Your object storage Access Key ID>
export AWS_SECRET_ACCESS_KEY=<Your object storage Access Key Secret>

s3cmd="aws --endpoint-url=$s3_url s3"
```

3. Next, let us create 2 buckets, one for storing the training data and another one for storing the training result.
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
  sed -i '' s/tf_training_data/$trainingDataBucket/ etc/examples/tf-model/manifest.yml
  sed -i '' s/tf_trained_model/$trainingResultBucket/ etc/examples/tf-model/manifest.yml
  sed -i '' s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/tf-model/manifest.yml
  sed -i '' s/user_name: test/user_name: $AWS_ACCESS_KEY_ID/ etc/examples/tf-model/manifest.yml
  sed -i '' s/password: test/password: $AWS_SECRET_ACCESS_KEY/ etc/examples/tf-model/manifest.yml
else
  sed -i s/tf_training_data/$trainingDataBucket/ etc/examples/tf-model/manifest.yml
  sed -i s/tf_trained_model/$trainingResultBucket/ etc/examples/tf-model/manifest.yml
  sed -i s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/tf-model/manifest.yml
  sed -i s/user_name: test/user_name: $AWS_ACCESS_KEY_ID/ etc/examples/tf-model/manifest.yml
  sed -i s/password: test/password: $AWS_SECRET_ACCESS_KEY/ etc/examples/tf-model/manifest.yml
fi
```

6. Now you should have all the necessary training data set in your training data bucket. Let's go ahead to set up your restapi endpoint
and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI (executable
binary).

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$PUBLIC_IP:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default TensorFlow model
CLI_CMD=cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```
