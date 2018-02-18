# Fabric for Deep Learning (FfDL) User Guide

## Table of Contents
1. [Supported Deep Learning Frameworks](#1-supported-deep-learning-frameworks)
2. [Create New Models with FfDL](#2-create-new-models-with-ffdl)
  * 2.1. [Model Definition Files](#21-model-definition-files)
  * 2.2. [Data Formatting](#22-data-formatting)
  * 2.3. [Uploading Data in Object Store](#23-uploading-data-in-object-store)
  * 2.4. [Creating Manifest file](#24-creating-manifest-file)
  * 2.5. [Creating Model zip file](#25-creating-model-zip-file)
  * 2.6. [Model Deployment and Training](#26-model-deployment-and-training)
    * 2.6.1. [Train models using FfDL CLI](#261-train-models-using-ffdl-cli)
    * 2.6.2. [Train models using FfDL UI](#262-train-models-using-ffdl-ui)
3. [Object Store for FfDL](#3-object-store-for-ffdl)
  * 3.1. [FfDL Local Object Store](#31-ffdl-local-object-store)
  * 3.2. [Cloud Object Store](#32-cloud-object-store)

### Prerequisites

* You need to have [FfDL](../README.md#5-detailed-installation-instructions) running on your cluster.

## 1. Supported Deep learning frameworks
Currently, Fabric for Deep Learning supports following frameworks

* [Tensorflow (with Keras 2)](https://www.tensorflow.org/) version "1.3-py3"
* [Caffe](http://caffe.berkeleyvision.org/) version "1.0-py2"

You can deploy models based on these frameworks and then train your models using the FfDL CLI or FfDL UI.

## 2. Create New Models with FfDL

To create new models you first need to create model definition files and data files for training and testing.

### 2.1. Model Definition Files

Different deep learning frameworks support different languages to define their models. For example, [Torch](http://torch.ch/) models are defined in [LuaJIT](http://luajit.org/) whereas [Caffe](http://caffe.berkeleyvision.org/) models are defined using config files written in [Protocol Buffer Language](https://developers.google.com/protocol-buffers/docs/proto). Details on how to write model definition files is beyond the scope of this document.

### 2.2. Data Formatting
Different frameworks require train and test datasets in different formats. For example, [Caffe requires datasets in LevelDB or LMDB format](http://caffe.berkeleyvision.org/tutorial/layers.html#data-layers) while Torch requires datasets in Torch proprietary format. We assume that data is already in the format needed by the specific framework. Details on how to convert raw data to framework specific format is beyond the scope of this document.  

### 2.3. Uploading Data in Object Store
Follow the instructions under [Object Store for FfDL](#3-object-store-for-ffdl). You can then use the object store credentials to upload your data. The object store is also used to store the trained model.

### 2.4. Creating Manifest file
The manifest file contains different fields describing the model in FfDL , its object store information, its resource requirements, and several arguments (including hyperparameters) required for model execution during training and testing.
Here are [example manifest files](../etc/examples/tf-model/manifest.yml) for Caffe and TensorFlow models. You can use these templates to create manifest file for your models. Below we describe different fields of the manifest file for FfDL.
* ```name:``` After a model is deployed in FfDL a unique id for the model is created. The model id is <name>+<mkey>, where <mkey> is a string of alphanumeric characters to uniquely identify the deployed model. <name> is a prefix of the model id. You can provide any value to name.
* ```version:``` This is version of the manifest file. This field is currently not used.
* ```description:``` This is for users to keep track of their deployed models. Users can use in future and get information about particular model. FfDL does not interpret it. You can put anything here.
* ```learners:``` Number of learners to use in training. As FfDL supports distributed learning, you can have more than one learner for your training job.
* ```gpus:``` Number of gpus used by each learner during training.
* ```cpus:``` Number of cpus used by each learner during training. The default cpu number is 8.
* ```memory:``` Memory assigned to each learner during training. The default memory is 60Gb.
* ```data_stores:```You can specify as many data stores as you want in the manifest file. Each data store has the following fields.
  * ```id:``` Data store id (**which you make up**), to be used when creating a training job.
  * ```type:``` Type of data store, values can be "swift_datastore", "s3_datastore", or "mount_cos" (details below).
  * ```training_data:``` Location of the training data in the data store.
    * ```container:``` The container where the training data resides.
  * ```training_results:``` Location of the training results in the data store. After training the trained model and training logs will be stored here, under "training-TRAININGJOBID".  
    * ```container:``` The container where the training results will be stored. **This filed is recommended to define by users. e.g. `mnist_trained_model`.** If not, the default location of trained models is FfDL object store.
  * ```connection:``` The connection variables for the data store. The list of connection variables supported is data store type dependent. At present, the following connection variables are supported:
    * swift_datastore (softlayer credential form): ```auth_url```, ```user_name```, and ```password```
    * swift_datastore (IBM Cloud credential form): ```auth_url```, ```user_name```, ```password```, ```domain_name```, ```region``` and ```project_id```
    * s3_datastore: ```auth_url```, ```user_name``` (AWS Access Key), and ```password``` (AWS Secret Access Key)
    * mount_cos: ```auth_url```, ```user_name``` (AWS Access Key), and ```password``` (AWS Secret Access Key), ```region``` (optional)

* ```framework:``` This field provides deep learning framework specific information.
  * ```name:``` Name of framework, values can be "caffe", or "tensorflow".
  * ```version:``` Version of framework.
  * ```command:``` This field identifies the main program file along with any arguments that FfDL needs to execute. For example, the command to run a TensorFlow training can be as follows ```python mnist_with_summaries.py --train_images_file ${DATA_DIR}/train-images-idx3-ubyte.gz --train_labels_file ${DATA_DIR}/train-labels-idx1-ubyte.gz --test_images_file ${DATA_DIR}/t10k-images-idx3-ubyte.gz --test_labels_file ${DATA_DIR}/t10k-labels-idx1-ubyte.gz --max_steps 400 --learning_rate 0.001``` where `python mnist_with_summaries.py` is the model code to execute while the remainder are arguments to the model. `train_images_file`, `train_labels_file`, `test_images_file`, `test_labels_file` refers to the dataset path in learner,  `max_steps`, `learning_rate` are training parameters and hyperparameters.

**Note**: If the user's model and manifest files refer to some training data, they shouldn't use absolute paths. They should either:

    * use relative paths, like

      --train_images_file ./train-images-idx3-ubyte.gz

      --test_images_file ./t10k-images-idx3-ubyte.gz

    * or, use the $DATA_DIR environment variable, like

      --train_images_file ${DATA_DIR}/train-images-idx3-ubyte.gz

      --test_images_file ${DATA_DIR}/t10k-images-idx3-ubyte.gz

### 2.5. Creating Model zip file
**Note** that FfDL CLI can take both zip or unzip files.

You need to zip all the model definition files and create a model zip file for jobs submitting on FfDL UI. At present, FfDL UI only supports zip format for model files, other compression formats like gzip, bzip, tar etc., are not supported. **Note** that all model definition files has to be in the first level of the zip file and there are no nested directories in the zip file.

### 2.6. Model Deployment and Training
After creating the manifest file and model definition file, you can either use the FfDL CLI or FfDL UI to deploy your model.

#### 2.6.1. Train models using FfDL CLI
> Note: Right now FfDL CLI only available on Mac and Linux.

In order to use the FfDL CLI, you will need your FfDL's restapi endpoint. Currently, the FfDL CLI is an executable binary located at `cli/bin`
```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;
```

Next, you need to get the correct executable binary for FfDL CLI located at `cli/bin`.
```shell
CLI_CMD=cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
```

Now, you can use the following command to train your models.
```shell
$CLI_CMD train <manifest file location>  <model definition zip | model definition directory>
```

After training your models, you can run `$CLI_CMD logs <Job ID>` to view your model's logs and `$CLI_CMD list` to view the list of models your had trained. You can also run `$CLI_CMD -h` to learn more about the FfDL CLI.

#### 2.6.2. Train models using FfDL UI
To train your models using FfDL UI, simply upload your manifest file and model definition zip in the correspond fields and click `Submit Training Job`

![ui-example](images/ui-example.png)

## 3. Object Store for FfDL
We will use the [Amazon's S3 command line interface](https://aws.amazon.com/cli/) to access the object store. To set up a user environment to access object store, please follow instructions at [AWS cli setup page](http://docs.aws.amazon.com/cli/latest/userguide/installing.html) and [Using Amazon S3 with the AWS cli](http://docs.aws.amazon.com/cli/latest/userguide/cli-s3.html).

### 3.1 FfDL Local Object Store
By default FfDL will use its local object store for storing any training and result data. You need to set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables using the default Local Object Store credentials before using this client.
```shell
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_DEFAULT_REGION=us-east-1

# Create your training data and result buckets
aws --endpoint-url=http://$(make --no-print-directory kubernetes-ip):$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}') s3 mb <trainingDataBucket>
aws --endpoint-url=http://$(make --no-print-directory kubernetes-ip):$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}') s3 mb <trainingResultBucket>
```

Now, upload all you datasets to the training data bucket.
```shell
aws --endpoint-url=http://$(make --no-print-directory kubernetes-ip):$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}') s3 cp <data file> s3://<trainingDataBucket>/<data file name>
```

### 3.2 Cloud Object Store
Provision an S3 based Object Storage from your Cloud provider. Take note of your Authentication Endpoints, Access Key ID and Secret.

> For IBM Cloud, you can provision an Object Storage from [IBM Cloud Dashboard](https://console.bluemix.net/catalog/infrastructure/cloud-object-storage?taxonomyNavigation=apps) or from [SoftLayer Portal](https://control.softlayer.com/storage/objectstorage).

You need to set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables using your Cloud Object Store credentials before using this client.

```shell
export AWS_ACCESS_KEY_ID=*****************
export AWS_SECRET_ACCESS_KEY=********************

# Create your training data and result buckets
aws --endpoint-url=http://<object storage Authentication Endpoints> s3 mb <trainingDataBucket>
aws --endpoint-url=http://<object storage Authentication Endpoints> s3 mb <trainingResultBucket>
```

Now, upload all you datasets to the training data bucket.
```shell
aws --endpoint-url=http://<object storage Authentication Endpoints> s3 cp <data file> s3://<trainingDataBucket>/<data file name>
```
