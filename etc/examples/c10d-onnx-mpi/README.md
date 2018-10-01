# PyTorch Fashion MNIST distributed training example using MPI and exported in ONNX

This is a 2 convolutional layers model using native pytorch method **dist.init_process_group** to sync and join all the worker nodes.
- Tested with Syncing using Open-MPI where worker processes are executed with mpirun command.
- Tested with **MPI** Backend.
- For more details on PyTorch distributed package, please refer to https://pytorch.org/tutorials/intermediate/dist_tuto.html

## Step 1 - Upload Fashion MNIST dataset and setup FfDL client

This assumes that you have access to the cluster and that you have provisioned FfDL already. To check if you have access to your cluster run the following command and if a table of information about the pods on your cluster appears then you have access.

```shell
kubectl get pods
```

Go to the [FfDL](https://github.com/IBM/FfDL) repository and define the necessary environment variables before you proceed to the next step.
```shell
cd <path to the FfDL repository>
export VM_TYPE=none
export PUBLIC_IP=<Cluster Public IP>
```

Setup your s3 command with the Object Storage credentials.
```bash
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$PUBLIC_IP:$s3_port
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;
s3cmd="aws --endpoint-url=$s3_url s3"
```

Create buckets to hold the training data and trained model
```shell
# Replace trainingdata with what you want the bucket holding the training data to be named
# Replace trainedmodel with what you want the bucket storing your model to be named
$s3cmd mb s3://mnist_pytorch
$s3cmd mb s3://mnist_trained_model
```

Run the below block of code in python to pre-process your data, then store them in your object storage bucket.
```python
from torchvision import datasets, transforms
dataset = datasets.FashionMNIST(
    "./data",
    train=True,
    download=True,
    transform=transforms.Compose([
        transforms.ToTensor(),
        transforms.Normalize((0.1307, ), (0.3081, ))
    ]))
```

Upload the processed Fashion MNIST training data to your training data bucket.
```shell
$s3cmd cp --recursive data s3://fashion_mnist/data
```

Now you should have all the necessary training data set in your training data bucket. Let's go ahead to set up your restapi endpoint and default credentials for Deep Learning as a Service. Once you done that, you can start running jobs using the FfDL CLI (executable binary).
```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$PUBLIC_IP:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
```

## Step 2 - Creating a Manifest File
Create the code that will train your model. We will use `train_dist_onnx_mpi.py` in the folder "model-files" as an example

Create a .yml file with the necessary information. manifest.yml has further instructions on creating an appropriate manifest file.

## Step 3a - Deploying the training job to FfDL

We will now go back to this directory and deploy our training job to FfDL using the path to the .yml and path to the folder containing the experiment.py
```bash
cd <path to this demo repo>/c10d-onnx-mpi
# Replace manifest.yml with the path to your .yml file
# Replace Image with the path to the folder containing your file created in step 6
$CLI_CMD train manifest.yml model-files
```
## Step 3b - Deploying the training job to FfDL using the FfDL UI

Alternatively, the FfDL UI can be used to deploy jobs. First zip your model.
```bash
# Replace fashion-training with the path to your training file folder
# Replace fashion-training.zip with the path where you want the .zip file stored
pushd model-file && zip ../fashion-data-parallel * && popd
```

Go to FfDL web UI. Upload the .zip to "Choose model definition zip to upload". Upload the .yml to "Choose manifest to upload". Then click Submit Training Job.

## Appropriate Test Data

Expects to receive a file path to a picture. Over 30 different file types are supported although only the two (.png and .jpg) have been tested extensively. These file types are listed at (http://pillow.readthedocs.io/en/5.1.x/handbook/image-file-formats.html)

The models trained on the Fashion MNIST data will work best when there is only one object in the picture and the background of the picture is pure black. Additionally, the object in the picture should be completely in frame.
