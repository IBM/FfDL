# Model Training and Serving using WML

## Prerequisite

Install [IBM Cloud CLI](https://console.bluemix.net/docs/cli/reference/bluemix_cli/get_started.html#getting-started) and [Machine Learning Plugin](). In addition setup your [AWS S3 command line](https://aws.amazon.com/cli/)

``` shell
bx plugin install ml_cli_plugin_osx
bx target -o ORG -s SPACE
``` 

# Steps

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

  - [1. Provision your WML instance](#1-provision-your-wml-instance)
    - [1.1 Create an instance of WML service and associated key using BX command line](#11-create-an-instance-of-wml-service-and-associated-key-using-bx-command-line)
    - [1.2 Get your service credentials](#12-get-your-service-credentials)
    - [1.3 Set the Machine Learning plugin it up with your creds obtained in step 2](#13-set-the-machine-learning-plugin-it-up-with-your-creds-obtained-in-step-2)
    - [1.4 Test your WML instance](#14-test-your-wml-instance)
  - [2. Provision an Object Storage Instance, and upload Training Data](#2-provision-an-object-storage-instance-and-upload-training-data)
  - [3. Create your Model Training Run](#3-create-your-model-training-run)
    - [3.1 Create Deep Learning Model Program and put them in a Zip file](#31-create-deep-learning-model-program-and-put-them-in-a-zip-file)
    - [3.2 Create a Training Run Manifest File](#32-create-a-training-run-manifest-file)
  - [4. Submit, Monitor and Store a Training Run](#4-submit-monitor-and-store-a-training-run)
    - [4.1 Submit](#41-submit)
    - [4.2 Monitor](#42-monitor)
    - [4.3 Save the Trained Model](#43-save-the-trained-model)
  - [5. Deploy and Serve Models](#5-deploy-and-serve-models)
    - [5.1 Deploy stored model to WML](#51-deploy-stored-model-to-wml)
    - [5.2 Score the deployed model.](#52-score-the-deployed-model)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## 1. Provision your WML instance


### 1.1 Create an instance of WML service and associated key using BX command line

``` shell
bx cf create-service pm-20 lite Animesh-WML
bx cf create-service-key Animesh-WML Animesh-WML-Key
``` 

### 1.2 Get your service credentials
``` shell
bx cf service-key Animesh-WML Animesh-WML-Key
```

### 1.3 Set the Machine Learning plugin it up with your creds obtained in step 2

``` shell
export ML_INSTANCE=11111111-aaaa-2222-bbbb-333333333333
export ML_USERNAME=44444444-cccc-5555-dddd-666666666666
export ML_PASSWORD=77777777-eeee-8888-ffff-999999999999
export ML_ENV=<url from credentials>
 ```
### 1.4 Test your WML instance

``` shell
AnimeshMacBook:~ animeshsingh$ bx ml list training-runs
Fetching the list of training runs ...
SI No   Name   guid   status   framework   version   submitted-at   

0 records found.
OK
List all training-runs successful
 ```
 
## 2. Provision an Object Storage Instance, and upload Training Data

Provision an [Object Storage instance](https://console.bluemix.net/catalog/services/cloud-object-storage). You then need to upload data in your Object storage. Here we are getting the data sets from [THE MNIST DATABASE of handwritten digits](http://yann.lecun.com/exdb/mnist/)

``` shell
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# Create your training data and result buckets
aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 mb <trainingDataBucket>
aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 mb <trainingResultBucket>

aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 cp t10k-labels-idx1-ubyte.gz s3://test-data-animesh/
aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 cp train-labels-idx1-ubyte.gz s3://test-data-animesh/
aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 cp t10k-images-idx3-ubyte.gz s3://test-data-animesh/
aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 cp train-images-idx3-ubyte.gz s3://test-data-animesh/

aws --endpoint-url=http://s3-api.us-geo.objectstorage.softlayer.net s3 ls s3://test-data-animesh
2018-03-10 00:14:49    1648877 t10k-images-idx3-ubyte.gz
2018-03-10 00:13:12       4542 t10k-labels-idx1-ubyte.gz
2018-03-10 00:15:22    9912422 train-images-idx3-ubyte.gz
2018-03-10 00:14:31      28881 train-labels-idx1-ubyte.gz
``` 

## 3. Create your Model Training Run
### 3.1 Create Deep Learning Model Program and put them in a Zip file

In this step we create a sample deep learning tensorflow program to train a model. For this, you must use the input_data.py and convolutional_network.py files, which you can find in the tf-model.zip file in this repository. This is for [THE MNIST DATABASE of handwritten digits](http://yann.lecun.com/exdb/mnist/)

In the convolutional_network.py file, there is one part which is important for IBM Watson Machine Learning service to score the model properly. The model should be trained to the RESULT_DIR/model directory after training is complete.

``` shell
zip tf-model.zip convolutional_network.py input_data.py

``` 

### 3.2 Create a Training Run Manifest File

Create a Training Run Manifest File. Please use the sample tf-train.yml from this repository. Make sure to point to your Object Storage instance
``` shell
model_definition:
  name: tf-mnist-showtest1
  author:
    name: DL Developer
    email: dl@example.com
  description: Simple MNIST model implemented in TF
  framework:
    name: tensorflow
    version: 1.2
  execution:
    command: python3 convolutional_network.py --trainImagesFile ${DATA_DIR}/train-images-idx3-ubyte.gz
      --trainLabelsFile ${DATA_DIR}/train-labels-idx1-ubyte.gz --testImagesFile ${DATA_DIR}/t10k-images-idx3-ubyte.gz
      --testLabelsFile ${DATA_DIR}/t10k-labels-idx1-ubyte.gz --learningRate 0.001
      --trainingIters 20000
    compute_configuration:
      name: small
training_data_reference:
  name: training_data_reference_name
  connection:
    endpoint_url: "https://s3-api.us-geo.objectstorage.service.networklayer.com"
    aws_access_key_id: "722432c254bc4eaa96e05897bf2779e2"
    aws_secret_access_key: "286965ac10ecd4de8b44306288c7f5a3e3cf81976a03075c"
  source:
    bucket: training-data
  type: s3
training_results_reference:
  name: training_results_reference_name
  connection:
    endpoint_url: "https://s3-api.us-geo.objectstorage.service.networklayer.com"
    aws_access_key_id: "722432c254bc4eaa96e05897bf2779e2"
    aws_secret_access_key: "286965ac10ecd4de8b44306288c7f5a3e3cf81976a03075c"
  target:
    bucket: training-results
  type: s3
``` 

## 4. Submit, Monitor and Store a Training Run

### 4.1 Submit

Submit Training Run
``` shell
bx ml train tf-model.zip tf-train.yaml
``` 

### 4.2 Monitor

Monitor Training Run

``` shell
bx ml list training-runs
bx ml show training-runs training-HrlzIHskg
``` 
Sample Output
``` shell
Fetching the training runs details with MODEL-ID 'training-HrlzIHskg' ...
ModelId        training-HrlzIHskg
url            /v3/models/training-HrlzIHskg
Name           tf-mnist-showtest1
State          running
Submitted_at   2017-11-17T17:01:39Z
OK
Show trained-runs details successful
``` 
To continously monitor the logs logs of Training Run

``` shell
bx ml monitor training-runs training-HrlzIHskg
```

When a training run has completed successfully (or failed) all files written to $RESULT_DIR and the logs from the run should be written to the Cloud Object Storage bucket specified in the setting training_results_reference within the training manifest file, under a folder with the same name as the model id.

### 4.3 Save the Trained Model

Once a training run has completed successfully, the trained model can be permanently stored into the repository from from where it can be later deployed for scoring. To do this use the command bx ml store training-runs <model-id>:

``` shell
bx ml store training-runs training-DOl4q2LkR
```
Sample Output:

``` shell
OK
Model store successful. Model-ID is '19db0ae7-3a9d-44e7-8e9d-fce3f4f8e0eb'.
```

You can inspect the trained model and logs in the object store. These appear in the training-HrlzIHskg folder in the test_results bucket.

You can list the files you have in "test_results"

``` shell
aws --endpoint-url=<ibm-cos-endpoint-url> --profile ibm_cos s3 ls s3://test_data/
training-HrlzIHskg/learner-1/load-data.log
training-HrlzIHskg/learner-1/load-model.log
training-HrlzIHskg/learner-1/training-log.txt
training-HrlzIHskg/model/saved_model.pb
training-HrlzIHskg/model/variables/variables.data-00000-of-00001
training-HrlzIHskg/model/variables/variables.index
training-HrlzIHskg/saved_model.tar.gz
You can download the saved model by running the command below

aws --endpoint-url=<ibm-cos-endpoint-url> --profile ibm_cos s3 cp s3://test_data/saved_model.tar.gz saved_model.tar.gz
``` 

## 5. Deploy and Serve Models

### 5.1 Deploy stored model to WML


``` shell
bx ml deploy a8379aaa-ea31-4c22-824d-89a01315dd6d "my_deployment"

Sample Output:

Deploying the model with MODEL-ID 'a8379aaa-ea31-4c22-824d-89a01315dd6d'...
DeploymentId       9d6a656c-e9d4-4d89-b335-f9da40e52179   
Scoring endpoint   https://2000ab8b-7e81-41b3-ad07-b70f849594f5.wml-fvt.ng.bluemix.net/v3/published_models/a8379aaa-ea31-4c22-824d-89a01315dd6d/deployments/9d6a656c-e9d4-4d89-b335-f9da40e52179/online   
Name               test34   
Type               tensorflow-1.2   
Runtime            python-3.5   
Created at         2017-11-28T12:46:19.770Z   
OK
Deploy model successful

```

### 5.2 Score the deployed model. 

To score the model, the scoring_payload.json file must use the following format:
``` shell
{"modelId": "a8379aaa-ea31-4c22-824d-89a01315dd6d","deploymentId": "9d6a656c-e9d4-4d89-b335-f9da40e52179","payload":{"inputs":[[0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0313725508749485,0.48235297203063965,0.6352941393852234,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.3294117748737335,0.40392160415649414,0.7921569347381592,0.8823530077934265,0.3333333432674408,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.07058823853731155,0.1568627506494522,0.6705882549285889,0.874509871006012,0.9411765336990356,0.9254902601242065,0.29019609093666077,0.22745099663734436,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.19215688109397888,0.250980406999588,0.5254902243614197,0.9411765336990356,1.0,0.9921569228172302,0.6313725709915161,0.22352942824363708,0.05490196496248245,0.1411764770746231,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.38431376218795776,0.8313726186752319,0.8352941870689392,0.9529412388801575,0.9921569228172302,0.9921569228172302,0.9921569228172302,0.7529412508010864,0.2666666805744171,0.08235294371843338,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.6274510025978088,0.9019608497619629,0.9058824181556702,0.9019608497619629,0.6235294342041016,0.3137255012989044,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.1411764770746231,0.3294117748737335,0.4745098352432251,0.10588236153125763,0.10588236153125763,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.16862745583057404,0.4941176772117615,0.3686274588108063,0.22352942824363708,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.3333333432674408,0.7921569347381592,0.1882353127002716,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.05490196496248245,0.9294118285179138,0.9529412388801575,0.13725490868091583,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.1411764770746231,0.7058823704719543,1.0,0.8235294818878174,0.7568628191947937,0.14509804546833038,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.125490203499794,0.24705883860588074,0.24705883860588074,0.7176470756530762,0.5568627715110779,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.5843137502670288,0.8000000715255737,0.03529411926865578,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.9960784912109375,0.8823530077934265,0.05490196496248245,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.9960784912109375,0.3686274588108063,0.01568627543747425,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.8627451658248901,0.04313725605607033,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.08627451211214066,0.6784313917160034,0.9960784912109375,0.24705883860588074,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.007843137718737125,0.4901961088180542,0.9921569228172302,0.8784314393997192,0.125490203499794,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.08627451211214066,0.9921569228172302,0.7843137979507446,0.1411764770746231,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.01568627543747425,0.5098039507865906,0.027450982481241226,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0],[0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.3607843220233917,0.9921569228172302,0.5529412031173706,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.07450980693101883,0.7686275243759155,0.988235354423523,0.9450981020927429,0.0941176563501358,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.5764706134796143,0.8823530077934265,0.3803921937942505,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.988235354423523,0.988235354423523,0.9921569228172302,0.10588236153125763,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.125490203499794,0.9921569228172302,0.988235354423523,0.7647059559822083,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.988235354423523,0.988235354423523,0.6980392336845398,0.03529411926865578,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.5490196347236633,0.9921569228172302,0.988235354423523,0.40000003576278687,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.988235354423523,0.988235354423523,0.5490196347236633,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.062745101749897,0.7960785031318665,0.9921569228172302,0.988235354423523,0.21568629145622253,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.4705882668495178,0.9921569228172302,0.9921569228172302,0.5529412031173706,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.11372549831867218,0.9921569228172302,1.0,0.8431373238563538,0.12156863510608673,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.7725490927696228,0.988235354423523,0.988235354423523,0.5490196347236633,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.11372549831867218,0.988235354423523,0.9921569228172302,0.16470588743686676,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.7725490927696228,0.988235354423523,0.988235354423523,0.12156863510608673,0.0,0.0,0.0,0.0,0.0,0.0,0.05098039656877518,0.30980393290519714,0.988235354423523,0.9921569228172302,0.10588236153125763,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.917647123336792,0.988235354423523,0.9254902601242065,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.988235354423523,0.988235354423523,0.6980392336845398,0.03529411926865578,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.3333333432674408,0.988235354423523,0.988235354423523,0.7411764860153198,0.5529412031173706,0.5490196347236633,0.5490196347236633,0.5490196347236633,0.5490196347236633,0.30980393290519714,0.1882353127002716,0.6470588445663452,0.988235354423523,0.988235354423523,0.5490196347236633,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.33725491166114807,0.9921569228172302,0.9921569228172302,0.9921569228172302,1.0,0.9921569228172302,0.9921569228172302,0.9921569228172302,0.9921569228172302,1.0,0.9921569228172302,0.9921569228172302,0.9921569228172302,0.9921569228172302,0.5529412031173706,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.22352942824363708,0.9137255549430847,0.988235354423523,0.988235354423523,0.9921569228172302,0.988235354423523,0.988235354423523,0.9490196704864502,0.8392157554626465,0.8431373238563538,0.9529412388801575,0.988235354423523,0.988235354423523,0.988235354423523,0.5490196347236633,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.29411765933036804,0.7647059559822083,0.7647059559822083,0.2196078598499298,0.21568629145622253,0.21568629145622253,0.19215688109397888,0.12156863510608673,0.12156863510608673,0.19607844948768616,0.8196079134941101,0.988235354423523,0.8627451658248901,0.12156863510608673,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.2862745225429535,0.917647123336792,0.988235354423523,0.4392157196998596,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.8823530077934265,0.988235354423523,0.988235354423523,0.4392157196998596,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.8862745761871338,0.9921569228172302,0.9921569228172302,0.4392157196998596,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.3960784673690796,0.9764706492424011,0.988235354423523,0.9490196704864502,0.29019609093666077,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.4431372880935669,0.988235354423523,0.988235354423523,0.9647059440612793,0.3450980484485626,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.14901961386203766,0.917647123336792,0.988235354423523,0.6039215922355652,0.38823533058166504,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.572549045085907,0.988235354423523,0.3294117748737335,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0,0.0]],"keep_prob":1.0}}

```

To score, run the following command, which passes the scoring_payload.json file to the scoring processor:

``` shell
bx ml score scoring_payload.json
```

Sample Output:

``` shelL
Fetching scoring results for the deployment 'e27c1fb7-0560-43df-bc9f-4c64580d67cd' ...
{"classes":[5,4]}
OK
Score request successful
```
