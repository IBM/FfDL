# PyTorch MNIST Classifier

# Train Model

Train the [pyTorch MNIST model](https://github.com/IBM/FfDL/tree/master/etc/examples/pytorch-model) following the steps in the [user guide](https://github.com/IBM/FfDL#6-detailed-testing-instructions).

# Wrap the Runtime Scorer
You can skip this step if you are happy to use the already packaged image ```seldonio/ffdl-pymnist``` from DockerHub.

The runtime MNIST scrorer is contained within a standalone [python class PyMnist.py](./PyMnist.py). This needs to be packaged in a Docker container to run within Seldon. For this we use [Redhat's Source-to-image](https://github.com/openshift/source-to-image).

 * Install [S2I](https://github.com/openshift/source-to-image#installation)
 * From the pytorch-model folder run the following s2i build. You will need to change *seldonio* to your Docker repo. You will need at least 8GB for your local Docker.
```
s2i build . seldonio/seldon-core-s2i-python2 seldonio/ffdl-pymnist:0.1
```
 * Push image to DockerHub or your Docker repo accessible from the FfDL cluster.

# Deploy Model
To deploy the model you need to create the deployment resource from the template ```ffdl-mnist-deployment.json```. You will need to set:

 * TRAINING_ID : Your FfDL Training ID
 * BUCKET_NAME : The name of the bucket containing your model

You will also need to create a kubernetes secret containing your bucket endpoint url, key and secret. You can use the ```bucket-secret.yaml``` as a template, e.g.:

 * edit ```bucket-secret.yaml```
 * Enter the base64 values for endpoint url, key, secret
    * On a linux system running bash shell you could do the following to get your values, ```echo -n "my key" | base64```
 * run ```kubectl create -f bucket-secret.yaml```

Create a deployment file by adding your settings, e.g. using sed below:

```
cat ffdl-mnist-deployment.json | sed 's/%TRAINING_ID%/training-84hIKJViR/' | sed 's/%BUCKET_NAME%/tf_trained_model/'  > ffdl-mnist-deployment_mydeploy.json
```

Deploy:
```
kubectl create -f ffdl-mnist-deployment_mydeploy.json
```

# Test

To test the running model with example MNIST images you can run either of two notebooks depending on how you exposed the Seldon APIs

 * [Ambassador Example](serving_ambassador.ipynb)
 * [Seldon OAuth Example](serving_oauth.ipynb)
 
