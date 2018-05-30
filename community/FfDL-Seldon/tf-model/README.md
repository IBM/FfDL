# TensorFlow MNIST Classifier

# Train Model

Train the [Tensorflow MNIST model](https://github.com/IBM/FfDL/tree/master/etc/examples/tf-model) following the steps in the [user guide](https://github.com/IBM/FfDL#6-detailed-testing-instructions).

# Wrap the Runtime Scorer
You can skip this step if you are happy to use the already packaged image ```seldonio/ffdl-mnist``` from DockerHub.

The runtime MNIST scrorer is contained within a standalone [python class TFMnist.py](./tf-model/TFMnist.py). This needs to be packaged in a Docker container to run within Seldon. For this we use [Redhat's Source-to-image](https://github.com/openshift/source-to-image).

 * Install [S2I](https://github.com/openshift/source-to-image#installation)
 * From the tf-model folder run, (change seldonio to your repo):
```
s2i build . seldonio/seldon-core-s2i-python2 seldonio/ffdl-mnist:0.1
```
 * Push image to DockerHub

# Deploy Model
To deploy the model you need to create the deployment resource from the template ffdl-mnist-deployment.json.tmpl. You will need to set:

 * TRAINING_ID : Your FfDL Training ID
 * BUCKET_KEY : The key for the bucket where your model is stored
 * BUCKET_URL : The URL for your Bucket

You will need to create a kubernetes secret containing your bucket secret. You can use the ```cat bucket-secret.yaml``` as a template, e.g.:

 * edit bucket-secret.yaml
 * Enter the base64 secret and save
 * run ```kubectl create -f bucket-secret.yaml```

Create a deployment file by adding your settings, e.g.

```
 cat ffdl-mnist-deployment.json | sed 's/%TRAINING_ID%/training-84hIKJViR/' | sed 's#%BUCKET_ENDPOINT_URL%#http://169.61.33.83:30537#' | sed 's/%BUCKET_KEY%/test/'  > ffdl-mnist-deployment_mydeploy.json
```

Deploy:
```
kubectl create -f ffdl-mnist-deployment_mydeploy.json
```

# Test

You will need to expose the API endpoints.






