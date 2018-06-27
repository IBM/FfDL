# Horovod TensorFlow Example

0. Deploy [FfDL](https://github.com/IBM/FfDL#5-detailed-installation-instructions) on your Kubernetes Cluster.

1. In the main FfDL repository, run the following commands to obtain the object storage endpoint from your cluster.
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
  sed -i '' s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/horovod/manifest_tfmnist.yml
else
  sed -i s/s3.default.svc.cluster.local/$node_ip:$s3_port/ etc/examples/horovod/manifest_tfmnist.yml
fi
```

Obtain the correct CLI for your machine and run the training job with our default Horovod model
```shell
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/horovod/manifest_tfmnist.yml etc/examples/horovod
```

Congratulations, you had submitted your first Horovod TensorFlow job on FfDL. You can check your FfDL status either from the FfDL UI or simply run `$CLI_CMD list`
