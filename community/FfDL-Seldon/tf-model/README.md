# TensorFlow MNIST Classifier

# Train Model

Train the [Tensorflow MNIST model](https://github.com/IBM/FfDL/tree/master/etc/examples/tf-model) following the steps in the [user guide](https://github.com/IBM/FfDL#6-detailed-testing-instructions).

# Wrap the Runtime Scorer
You can skip this step if you are happy to use the already packaged image ```seldonio/ffdl-mnist``` from DockerHub.

The runtime MNIST scrorer is contained within a standalone [python class TFMnist.py](./tf-model/TFMnist.py). This needs to be packaged in a Docker container to run within Seldon. For this we use [Redhat's Source-to-image](https://github.com/openshift/source-to-image).

 * Install [S2I](https://github.com/openshift/source-to-image#installation)
 * From the tf-model folder run, (*change seldonio to your Docker repo*):
```
s2i build . seldonio/seldon-core-s2i-python2 seldonio/ffdl-mnist:0.1
```
 * Push image to DockerHub or your Docker repo accessible from the FfDL cluster.

# Deploy Model
To deploy the model you need to create the deployment resource from the template ```ffdl-mnist-deployment.json```. You will need to set:

 * TRAINING_ID : Your FfDL Training ID
 * BUCKET_KEY : The key for the bucket where your model is stored
 * BUCKET_URL : The URL for your Bucket

You will need to create a kubernetes secret containing your bucket secret. You can use the ```bucket-secret.yaml``` as a template, e.g.:

 * edit ```bucket-secret.yaml```
 * Enter the base64 secret and save
 * run ```kubectl create -f bucket-secret.yaml```

Create a deployment file by adding your settings, e.g. using sed below:

```
 cat ffdl-mnist-deployment.json | sed 's/%TRAINING_ID%/training-84hIKJViR/' | sed 's#%BUCKET_ENDPOINT_URL%#http://1.2.3.4:30537#' | sed 's/%BUCKET_KEY%/test/'  > ffdl-mnist-deployment_mydeploy.json
```

Deploy:
```
kubectl create -f ffdl-mnist-deployment_mydeploy.json
```

# Test

To test the running model with example MNIST images you can run either of two notebooks depending on how you exposed the Seldon APIs

 * [Ambassador Example](serving_ambassador.ipynb)
 * [Seldon OAuth Example](serving_oauth.ipynb)
 
