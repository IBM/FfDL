# PyTorch MNIST Classifier

# Assumptions

 * You have installed [seldon-core](https://github.com/SeldonIO/seldon-core/blob/master/docs/install.md) on you FfDL cluster.


# Train Model

Train the [pyTorch MNIST model](../../../etc/examples/c10d-dist-onnx) following the steps in the [user guide](https://github.com/IBM/FfDL/blob/master/docs/detailed-installation-guide.md#2-detailed-testing-instructions).

# Wrap the Runtime Scorer
You can skip this step if you are happy to use the already packaged image ```ffdl/ngraph-onnx:0.1``` from DockerHub.

The runtime MNIST scorer is contained within a standalone [python class ONNXMnist.py](./ONNXMnist.py). This needs to be packaged in a Docker container to run within Seldon. For this we use [Redhat's Source-to-image](https://github.com/openshift/source-to-image).

 * Install [S2I](https://github.com/openshift/source-to-image#installation)
 * From the pytorch-model folder run the following s2i build. You will need to change *seldonio* to your Docker repo.
```
s2i build . seldonio/seldon-core-s2i-python3-ngraph-onnx:0.1 <username>/ngraph-onnx:0.1
```
 * Push image to DockerHub or your Docker repo accessible from the FfDL cluster.

# Deploy Model
To deploy the model you need to create the deployment resource from the template ```fashion-seldon-mount.json```. You will need to set:

 * TRAINING_ID : Your FfDL Training ID
 * BUCKET_NAME : The name of the bucket containing your model

You will also need to create a kubernetes secret containing your bucket endpoint url, key and secret. You can use the ```bucket-secret.yaml``` as a template, e.g.:

 * edit ```bucket-secret.yaml```
 * Enter the base64 values for key and secret
    * On a linux system running bash shell you could do the following to get your values, ```echo -n "my key" | base64```
 * run ```kubectl create -f bucket-secret.yaml```

Create a deployment file by adding your settings, e.g. using sed below:

```
cat fashion-seldon-mount.json | sed 's/%TRAINING_ID%/training-9qLhEIpmR/' | sed 's/%BUCKET_NAME%/tf_trained_model/' | sed 's/%OBJECT_STORAGE_ENDPOINT%/http://s3-api.us-geo.objectstorage.softlayer.net/'  > fashion-seldon-mount-mydeploy.json
```

Deploy:
```
kubectl create -f fashion-seldon-mount-mydeploy.json
```
