# Build the FfDL Fashion MNIST Web App and push it on Kubernetes

The webapp is designed to take images that are uploaded, display them on the webpage, and show the names of the classes with the top 3 confidences. There is an accompanying word cloud where the size of the word is correlated to the frequency of the class being the number one choice for a picture.

1. Go to the `fashion-mnist-webapp` directory, run the following commands to containerize your web application and push it to DockerHub.
  ```shell
  cd fashion-mnist-webapp-onnx
  docker build -t <your web app image name> .
  docker push <your web app image name>
  ```

2. Modify deployment resource from the template `fashion-mnist-webapp.yaml`. You need to set:
  * `MODEL_ENDPOINT`: Your Seldon model endpoint. (e.g. http://<AMBASSADOR_API_IP>/seldon/<Model_Deployment_Name>/api/v0.1/predictions) The `AMBASSADOR_API_IP` is your `seldon-core-ambassador`'s service endpoint which by default is exposed with NodePort.
  * `image` : Your web app image at DockerHub

3. Congratulations, your web app should be running now. You can use the following commands to check where your web app is hosted.
  ```shell
  kubectl get svc fashion-mnist-webapp-onnx
  # Your web app link is http://<Load_Balancer_IP>:8088
  ```

## Reference
This web-app example is based on the Model Asset Exchange web-app example. (https://github.com/IBM/MAX-Image-Caption-Generator-Web-App)
