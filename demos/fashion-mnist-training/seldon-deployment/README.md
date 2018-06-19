# Deploy your Fashion MNIST model with Seldon

[Seldon](https://github.com/SeldonIO/seldon-core) is an open source platform for deploying machine learning models on Kubernetes. Since FfDL stores your trained model in your Object Storage, it's very simple to deploy and serve it using Seldon with the following steps.

1. Install Seldon(https://github.com/SeldonIO/seldon-core/blob/master/docs/install.md) on your Kubernetes Cluster. Then, install S2I(https://github.com/openshift/source-to-image#installation) for building any Seldon model image.

2. Go to the `seldon-deployment` directory, build your Seldon model image using S2I(https://github.com/openshift/source-to-image#installation) and push it to DockerHub.
  ```shell
  cd seldon-deployment
  s2i build . seldonio/seldon-core-s2i-python2 <your model image name>
  docker push <your model image name>
  ```

3. Modify deployment resource from the template `fashion-seldon.json`. You need to set:
  * `TRAINING_ID` : Your FfDL Training ID
  * `BUCKET_NAME` : The name of the bucket containing your model
  * `image` : Your Seldon model image at DockerHub

  You also need to create a kubernetes secret containing your bucket endpoint url, key and secret. You can use the `bucket-secret.yaml` as a template. Edit `bucket-secret.yaml` and enter the base64 values for endpoint url, key, secret. (e.g. `echo -n "my key" | base64`)

  Then, deploy the `bucket-secret.yaml` to your cluster.
  ```shell
  kubectl apply -f bucket-secret.yaml
  ```

4. Deploy your Seldon model image on your Kubernetes Cluster.
  ```shell
  kubectl apply -f fashion-seldon.json
  ```

Congratulations, your model is now ready to take input and return predictions of confidence in the picture being of a certain class.
