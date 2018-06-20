# Perform Automated Machine Learning With H2O on FfDL

[H2O.ai](https://h2o.ai) provides an open source platform for automated machine learning: [H2O-3](https://www.h2o.ai/h2o/)

H2O is an open source, in-memory, distributed, fast, and scalable machine learning and predictive analytics platform that allows you to build machine learning models on big data and provides easy productionalization of those models in an enterprise environment.

![ffdl-h20](images/ffdl-h20.png)

# Deployment Steps

1. Follow steps to deploy FfDL from the [user guide](https://github.com/IBM/FfDL/blob/master/docs/user-guide.md)
2. Add some data, either follow the user guide to store the data locally or host the data in a cloud storage bucket and pull it at runtime.
3. Change the manifest.yaml to the settings that you want
  * NOTE: It is recommended that you allocate at least 4x the amount of memory as the size of the dataset you are trying to run H2O with.
  * EXAMPLE: 1.5GB Dataset --> 6.0 GB Memory allocated
4. Once FfDL is deployed in your Kubernetes cluster, use the CLI or GUI to deploy H2O

# Examples
Sample deployment scripts are hosted under: FfDL/community/FfDL-H2Oai

If you need a sample dataset, you can pull this toy dataset:

Train Set:
https://s3.amazonaws.com/h2o-public-test-data/smalldata/higgs/higgs_train_10k.csv

Test Set:
https://s3.amazonaws.com/h2o-public-test-data/smalldata/higgs/higgs_test_5k.csv

# Deploy H2O example on FfDL

0. Deploy [FfDL](https://github.com/IBM/FfDL#5-detailed-installation-instructions) on your Kubernetes Cluster.

1. In the main FfDL repository, run the following commands to obtain the object storage endpoint from your cluster.
```shell
node_ip=$(make --no-print-directory kubernetes-ip)
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$node_ip:$s3_port
```

2. Next, set up the default object storage access ID and KEY. Then create buckets for all the necessary training data and models.
```shell
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;

s3cmd="aws --endpoint-url=$s3_url s3"
$s3cmd mb s3://h2o3_training_data
$s3cmd mb s3://h2o3_trained_model
```

3. Now, create a temporary repository, download the necessary data for training the H2O model, and upload those data
to your h2o3_training_data bucket.

```shell
mkdir tmp
test -e tmp/higgs_train_10k.csv || wget -q -O tmp/higgs_train_10k.csv https://s3.amazonaws.com/h2o-public-test-data/smalldata/higgs/higgs_train_10k.csv
$s3cmd cp tmp/higgs_train_10k.csv s3://tf_training_data/higgs_train_10k.csv
```

4. Now you should have all the necessary training data set in your object storage. Let's go ahead to set up your restapi endpoint
and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI (executable
binary).

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default H2O model
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train community/FfDL-H2Oai/h2o-model/manifest-h2o.yml community/FfDL-H2Oai/h2o-model
```

Congratulations, you had submitted your first H2O job on FfDL. You can check your FfDL status either from the FfDL UI or simply run `$CLI_CMD list`
