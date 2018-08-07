# Minikube
## Deploy
* If `make deploy` dies after "Initializing...", most likely `VM_TYPE=minikube` was not set.
* If `make deploy` gets stuck at "Installing helm/tiller...", most likely helm is not installed.

## Training
* GPU image when only CPU available:

```
Deploying model with manifest '<manifest>' and model files in '<folder>/'...
FAILED
Error 200: OK
```

* Manifest not in zip file

```
FAILED
Error opening manifest file <folder>/<manifest>: open <folder>/<manifest>: no such file or directory

FAILED
Error reading manifest file.
```

* FfDL image names differ from DLaaS

```
Deploying model with manifest '<manifest>' and model files in '<folder>/'...

FAILED
Error: tensorflow version 1.3-py3 not supported.

FAILED
Error 200: OK
```
Need to adapt tensorflow version in manifest to what is specified on https://github.com/IBM/FfDL/blob/master/docs/user-guide.md#1-supported-deep-learning-frameworks (in this case "1.3.0-py3")

# DIND
## Deploy
* ffdl-lcm, ffdl-restapi, ffdl-trainer, ffdl-trainingdata and ffdl-ui pods show ImagePullBackOff: See if Kubernetes secret regcred exists via `kubectl get secret | grep regcred`. If it does not (output empty), create it with `kubectl create secret docker-registry regcred --docker-server=${DOCKER_REPO} --docker-username=${DOCKER_REPO_USER} --docker-password=${DOCKER_REPO_PASS} --docker-email=unknown@docker.io -n ${NAMESPACE}`.

## Training
* If you start a job and `lhelper` and `jobmonitor` pods get to `Running` state, but the corresponding `learner` remains stuck in `ContainerCreating`, please take a look at `kubectl describe pod <learner-pod>`. It is possible that your storage configuration in your manifest is invalid and if so, you should see events that point out the issues.
