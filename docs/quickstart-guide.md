# Quick Start Guide for FfDL

*NOTE: THIS GUIDE IS EXPERIMENTAL. FOLLOW AT YOUR OWN RISK*

This quick start guide for FfDL describes how to setup FfDL on a baremetal instance.
Starting from setting up a user, Docker and Kubernetes we will setup S3 drivers, build and install FfDL and finally run an exemplary training job. 

## Kick Start on IBM Cloud

### Set environment
```
kubectl create secret docker-registry regcred --docker-username <registry_username> --docker-password <registry_password> --docker-server registry.ng.bluemix.net --docker-email <registry_email>
export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold"
export PUBLIC_IP=<IP_TO_CLUSTER>
export DOCKER_REPO_USER=<REPOSITORY_USER>
export DLAAS_IMAGE_PULL_POLICY=Always
export DOCKER_PULL_POLICY=Always
export DOCKER_REPO_PASS=<PASSWORD_TO_YOUR_REPOSITORY>
export DOCKER_NAMESPACE=<NAMESPACE_ON_IBM_CLOUD>
export DOCKER_REPO=registry.ng.bluemix.net
export CLUSTER_NAME=<IBM_CLOUD_CLUSTER_NAME>
export VM_TYPE=none
export HAS_STATIC_VOLUMES=True
```

### Build
```
glide install
make build gen-certs docker-build docker-push
```

### Deploy & Test
```
cd bin
./create_static_volumes.sh
./create_static_volumes_config.sh
# Wait while kubectl get pvc shows static-volume-1 in state Pending
cd ..
make deploy
make test-submit
```

## Kick Start on DIND

If you have no Kubernetes cluster, yet, please start with the next section.

```
# Start Kubernetes Cluster
./dind-cluster-v1.9.sh up
# Start Docker Registry
cd ~/docker-registry
docker-compose up -d
# Launch S3
./launchs3driver.sh
export VM_TYPE=none
export PUBLIC_IP=10.192.0.3
export DOCKER_REPO_USER=<registry_user>
export DOCKER_REPO_PASS=<registry_password>
export DOCKER_REPO=<registry_url>
# Launch FfDL
cd $GOPATH/src/github.com/IBM/FfDL
make build docker-build docker-push create-volumes deploy
# Run training job
cd etc/examples/
DLAAS_URL=http://10.192.0.3:$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}') DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux train tf-model/manifest.yml tf-model
```

where PUBLIC_IP belongs to the node the REST API runs on. It can be found via `kubectl describe pod ffdl-restapi-...` (look at NodeIP value).

### Setup DIND on fresh SoftLayer instance
In order to setup FfDL with all dependencies on a fresh SoftLayer instance, please execute `bin/dind_scripts/experimental_master.sh`.
It is advisable to run it in your current shell, i.e. `. experimental_master.sh`.

### Build and Deploy

Build FfDL source: `make build`

Build FfDL Docker containers: `make docker-build`

Deploy FfDL to Kubernetes: `make deploy`

[To stop: `make undeploy`]

### Automatic Training
Push data to local S3 with
`make test-push-data`

Run with
`make test-submit`

### Manual Training
The following steps assume you have uploaded the training data to S3, e.g. via `make test-push-data`

Start and follow a training job:

`DLAAS_URL=http://10.192.0.3:30132 DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux train manifest.yml .`

`DLAAS_URL=http://10.192.0.3:30132 DLAAS_USERNAME=test-user DLAAS_PASSWORD=test $GOPATH/src/github.com/IBM/FfDL/cli/bin/ffdl-linux logs --follow training-...`

View S3 content in localstack:
`AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws s3 ls --endpoint-url http://10.192.0.4:32083 s3://tf_training_data`

Note that these commands use two distinct URLs for the FfDL REST API as well as S3. They might differ in your case and can be obtained as follows.

The former (i.e. FfDL REST API) uses
`$PUBLIC_IP:$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')`

whereas the latter (i.e. localstack S3 instance) uses
`$(kubectl get po/storage-0 -o=jsonpath='{.status.hostIP}'):$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')`

Alternatively, we could also extract the S3 IP via `sed`:
```
IP=`echo "$(kubectl describe po/storage-0)" | sed -r -n -e '/^Node:/p' | sed 's/^Node:.*\/\(.*\)/\1/'`
```

### Clone FfDL
One all prerequisites are installed, please clone FfDL to `$GOPATH/src/github.com/IBM/FfDL/`
and then proceed with the kickstart instructions.

## References

### FfDL
- [FfDL Repository on Github](https://github.com/IBM/FfDL)
- [FfDL IBM Code Open Project Page](https://developer.ibm.com/code/open/projects/fabric-for-deep-learning-ffdl/)
- [developerWorks: Fabric for Deep Learning](https://developer.ibm.com/code/2018/03/20/fabric-for-deep-learning/)
- [developerWorks: Democratize AI with Fabric for Deep Learning (FfDL)](https://developer.ibm.com/code/2018/03/20/democratize-ai-with-fabric-for-deep-learning/)
- [Introducing Fabric for Deep Learning (FfDL) on medium.com](https://medium.com/ibm-watson/introducing-fabric-for-deep-learning-ffdl-542522774775)
- [Q&A on IBM's Fabric for Deep Learning with Chief Architect of Watson](https://www.infoq.com/news/2018/04/ffdl-ruchir-puri)

### Watson Machine Learning and Watson Studio
- [IBM Watson Machine Learning](https://www.ibm.com/cloud/machine-learning)
- [Deep Learning as a Service with Watson Studio](https://www.ibm.com/cloud/deep-learning)
- [IBM Watson Studio](https://www.ibm.com/cloud/watson-studio)
- [IBM Watson Studio Workshop](https://github.com/PubChimps/DLaaSWorkshop)

### Related IBM Projects
- [Adversarial Robustness Toolbox](https://www.ibm.com/blogs/research/2018/04/ai-adversarial-robustness-toolbox/)
- [Seq2Seq-Vis](http://seq2seq-vis.io/)
- [LSTMVis](https://arxiv.org/abs/1606.07461)

### Kubernetes
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kubernetes DIND Cluster](https://github.com/Mirantis/kubeadm-dind-cluster)
- [developerWorks: Setup GPU-Enabled Kubernetes](https://developer.ibm.com/code/howtos/k8s-kubeadm-gpu-setup)
- [Nvidia Device Plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin)
- [IBM Cloud Container Service](https://www.ibm.com/cloud/container-service)

### localstack, IBM Cloud Object Storage and AWS S3 CLI
- [localstack - a fully functional AWS cloud stack](https://github.com/localstack/localstack)
- [IBM Cloud Object Storage](https://www.ibm.com/cloud/object-storage)
- [AWS S3 CLI Documentation](https://docs.aws.amazon.com/cli/latest/userguide/using-s3-commands.html)

### Papers
- [IBM Deep Learning Service](https://arxiv.org/abs/1709.05871)
- [Scalable Multi-Framework Multi-Tenant Lifecycle Management of Deep Learning Training Jobs](http://learningsys.org/nips17/assets/papers/paper_29.pdf)

### Videos
- [Introduction to Fabric for Deep Learning (FfDL)](https://www.youtube.com/watch?v=aKOqFL7VWhI)
- [Fabric for Deep Learning](https://www.youtube.com/watch?v=nQsYWmkfLP4)
- [FfDL at IEEE Computer Society Silicon Valley](https://youtu.be/wPKin0mN9LA?t=743)

### Slides
- [Fabric for Deep Learning](https://www.slideshare.net/AnimeshSingh/fabric-for-deep-learning-94941117)

## Addendum

### Relevant S3 Commands

Empty COS bucket

`aws s3 rm --recursive --endpoint-url https://s3-api.us-geo.objectstorage.softlayer.net s3://<bucket_name>`

List COS bucket content

`aws s3 ls --endpoint-url https://s3-api.us-geo.objectstorage.softlayer.net s3://<bucket_name>`

Copy file to COS (also works to download COS files if parameters are flipped)

`aws s3 cp --endpoint-url https://s3-api.dal-us-geo.objectstorage.service.networklayer.com <local_file> s3://<target_bucket>`

Upload all files from current directory to COS

`aws s3 cp --endpoint-url https://s3-api.dal-us-geo.objectstorage.service.networklayer.com --recursive . s3://<target_bucket>`

### Installation of Nvidia Docker
Nvidia Docker can be installed as follows (assuming Nvidia drivers are present):
```
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | \
  sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/ubuntu16.04/amd64/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list
sudo apt-get update
sudo apt-get install -y nvidia-docker2
sudo pkill -SIGHUP dockerd
docker run --runtime=nvidia --rm nvidia/cuda nvidia-smi
```

### Other useful commands
Delete old training artifacts:
`kubectl delete pod,pvc,pv,sc,deploy,svc,statefulset,secrets --selector training_id`

Get additional logs for debugging:
`kubectl logs po/kube-controller-manager-kube-master -n kube-system`
