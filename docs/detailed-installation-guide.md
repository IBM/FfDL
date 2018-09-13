
# Detailed Installation Guide


1. [Detailed Installation Instructions](#5-detailed-installation-instructions)
2. [Detailed Testing Instructions](#6-detailed-testing-instructions)
  - 2.1 [Using FfDL Local S3 Based Object Storage](#61-using-ffdl-local-s3-based-object-storage)
  - 2.2 [Using Cloud Object Storage](#62-using-cloud-object-storage)

## 1. Detailed Installation Instructions

0. If you don't have a Kubernetes Cluster, you can create a [Kubeadm-DIND](https://github.com/kubernetes-sigs/kubeadm-dind-cluster#using-preconfigured-scripts) Kubernetes Cluster on your local machine. We recommend you give at least 4 CPUs and 8GB of memory to your Docker.
> For Mac users, visit the instructions on the [Docker website](https://docs.docker.com/docker-for-mac/#advanced) and learn how to give more memory to your Docker.

1. First, clone this repository and install the helm tiller on your Kubernetes cluster.
``` shell
helm init

# Make sure the tiller pod is Running before proceeding to the next step.
kubectl get pods --all-namespaces | grep tiller-deploy
# kube-system   tiller-deploy-fb8d7b69c-pcvc2              1/1       Running
```

2. Define the necessary environment variables.
> If you are using bash shell, you can modify the necessary environment variables in `env.txt` and export all of them using the following commands
>  ```shell
>  source env.txt
>  export $(cut -d= -f1 env.txt)
>  ```

  * 2.a. For Kubeadm-DIND Cluster only
  ```shell
  export FFDL_PATH=$(pwd)
  export SHARED_VOLUME_STORAGE_CLASS=""
  export VM_TYPE=dind
  export PUBLIC_IP=localhost
  export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before proceeding to the next step
  ```

  * 2.b. For Cloud Kubernetes Cluster
  > Note: If you are using IBM Cloud Cluster, you can obtain your k8s public ip using `bx cs workers <cluster-name>`.

  ```shell
  # Change the storage class to what's available on your Cloud Kubernetes Cluster.
  export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold"
  export VM_TYPE=none
  export PUBLIC_IP=<Cluster Public IP>
  export NAMESPACE=default # If your namespace does not exist yet, please create the namespace `kubectl create namespace $NAMESPACE` before proceeding to the next step
  ```

3. Install the Object Storage driver using helm install.
  * 3.a. For Kubeadm-DIND Cluster only
  ```shell
  ./bin/s3_driver.sh
  helm install storage-plugin --set dind=true,cloud=false,namespace=$NAMESPACE
  ```

  * 3.b. For Cloud Kubernetes Cluster
  ```shell
  helm install storage-plugin --set namespace=$NAMESPACE
  ```

4. Create a static volume to store any metadata from FfDL.

```shell
pushd bin
./create_static_volumes.sh
./create_static_volumes_config.sh
# Wait while kubectl get pvc shows static-volume-1 in state Pending
popd
```

5. Now let's install all the necessary FfDL components using helm install.

``` shell
helm install . --set lcm.shared_volume_storage_class=$SHARED_VOLUME_STORAGE_CLASS,namespace=$NAMESPACE
```
> Note: If you want to upgrade an older version of FfDL, run
> `helm upgrade $(helm list | grep ffdl | awk '{print $1}' | head -n 1) .`

Make sure all the FfDL components are installed and running before moving to the next step.
``` shell
kubectl config set-context $(kubectl config current-context) --namespace=$NAMESPACE
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

6. Obtain the necessary port for Grafana, FfDL Web UI, local object storage, and FfDL restapi.
```shell
grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
ui_port=$(kubectl get service ffdl-ui -o jsonpath='{.spec.ports[0].nodePort}')
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
```

* For Kubeadm-DIND Cluster, we need to run the below script to forward the port to the localhost machine since we don't want to exec into the docker image and install various dependencies.
  ```shell
  ./bin/dind-port-forward.sh
  ```

7. Run the following commands to configure Grafana for monitoring FfDL using the logging information from prometheus.
  ```shell
  ./bin/grafana.init.sh
  ```

8. Lastly, run the following commands to obtain your Grafana, FfDL Web UI, and FfDL restapi endpoints.
``` shell
# Note: $(make --no-print-directory kubernetes-ip) simply gets the Public IP for your cluster.
node_ip=$PUBLIC_IP

# Echo statements to print out Grafana and Web UI URLs.
echo "Monitoring dashboard: http://$node_ip:$grafana_port/ (login: admin/admin)"
echo "Web UI: http://$node_ip:$ui_port/#/login?endpoint=$node_ip:$restapi_port&username=test-user"
```

Congratulation, FfDL is now running on your Cluster. Now you can go to [Step 6](#6-detailed-testing-instructions) to run some sample jobs or go to the [user guide](docs/user-guide.md) to learn about how to run and deploy your custom models.

## 2. Detailed Testing Instructions

In this example, we will run some simple jobs to train a convolutional network model using TensorFlow and Caffe. We will download a set of
MNIST handwritten digit images, store them with Object Storage, and use the FfDL CLI to train a handwritten digit classification model.

### 2.1. Using FfDL Local S3 Based Object Storage

1. Run the following commands to obtain the object storage endpoint from your cluster.
```shell
node_ip=$PUBLIC_IP
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$node_ip:$s3_port
```

2. Next, set up the default object storage access ID and KEY. Then create buckets for all the necessary training data and models.
```shell
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;

s3cmd="aws --endpoint-url=$s3_url s3"
$s3cmd mb s3://tf_training_data
$s3cmd mb s3://tf_trained_model
$s3cmd mb s3://mnist_lmdb_data
$s3cmd mb s3://dlaas-trained-models
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
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;
```

Replace the default object storage path with your s3_url. You can skip this step if your already modified the object storage path with your s3_url.
```shell
if [ "$(uname)" = "Darwin" ]; then
  sed -i '' s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/tf-model/manifest.yml
else
  sed -i s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/tf-model/manifest.yml
fi
```

Define the FfDL command line interface and run the training job with our default TensorFlow model
```shell
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

Congratulations, you had submitted your first job on FfDL. You can check your FfDL status either from the FfDL UI or simply run `$CLI_CMD list`

> You can learn about how to create your own model definition files and `manifest.yaml` at [user guild](docs/user-guide.md#2-create-new-models-with-ffdl).

5. If you want to run your job via the FfDL UI, simply run the below command to create your model zip file.

```shell
# Replace tf-model with the model you want to zip
pushd etc/examples/tf-model && zip ../tf-model.zip * && popd
```

Then, upload `tf-model.zip` and `manifest.yml` (The default TensorFlow model) in the `etc/examples/` repository as shown below.
Then, click `Submit Training Job` to run your job.

![ui-example](docs/images/ui-example.png)

6. (Optional) Since it's simple and straightforward to submit jobs with different deep learning framework on FfDL, let's try to run a Caffe Job. Download all the necessary training and testing images in [LMDB format](https://en.wikipedia.org/wiki/Lightning_Memory-Mapped_Database) for our Caffe model
and upload those images to your mnist_lmdb_data bucket.

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

7. Now train your Caffe Job.

```shell
$CLI_CMD train etc/examples/caffe-model/manifest.yml etc/examples/caffe-model
```

Congratulations, now you know how to deploy jobs with different deep learning framework. To learn more about your job execution results,
you can simply run `$CLI_CMD logs <MODEL_ID>`

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
