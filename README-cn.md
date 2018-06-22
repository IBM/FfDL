*阅读本文的其他语言版本：[English](README.md)。*

[![构建状态](https://travis-ci.org/IBM/FfDL.svg?branch=master)](https://travis-ci.org/IBM/FfDL)

# Fabric for Deep Learning (FfDL)

该存储库包含 *FfDL* (Fabric for Deep Learning) 平台的核心服务。FfDL 是深度学习的操作系统“结构”

FfDL 是一种协作平台，用于实现以下目的：
- 在分布式硬件上独立于框架对深度学习模型进行训练
- 开放深度学习 API  
- 提供通用度量工具
- 运行在用户的私有云或公共云中托管的深度学习

![ffdl-architecture](docs/images/ffdl-architecture.png)

要了解有关架构详情的更多信息，请阅读[此处](design/design_docs.md)

## 前提条件

* `kubectl`：Kubernetes 命令行界面 (https://kubernetes.io/docs/tasks/tools/install-kubectl/)

* `helm`：Kubernetes 软件包管理器 (https://helm.sh)

* `docker`：Docker 命令行界面 (https://www.docker.com/)

* `S3 CLI`：用于配置对象存储的[命令行界面](https://aws.amazon.com/cli/)

* 一个现有的 Kubernetes 集群（例如，进行本地测试的 [Minikube](https://github.com/kubernetes/minikube)）。对于 Minikube，使用 `make minikube` 命令来启动 Minikube，并设置本地网络路由。Minikube **v0.25.1** 已经过 Travis CI 的测试。

* 遵照相应的操作说明，使用 [IBM Cloud Public](https://github.com/IBM/container-journey-template/blob/master/README.md) 或 [IBM Cloud Private](https://github.com/IBM/deploy-ibm-cloud-private/blob/master/README.md) 支持 Kubernetes 集群

* FfDL 的最小推荐容量为 4GB 内存和 3 个 CPU。

## 使用方案

* 如果您已经快速入门并熟悉运用 FfDL 部署，便可跳至 [FfDL 用户指南](docs/user-guide.md)，利用 FfDL 训练深度学习模型。

* 如果您已配置 FfDL 来使用 GPU，并且希望利用 GPU 进行训练，请遵照[此处](docs/gpu-guide.md)的这些步骤

* 如果您已使用 FfDL 来训练模型，并且希望使用支持 GPU 的公共云托管服务进行进一步的训练和维护，请遵照此处的[操作说明](etc/converter/ffdl-wml.md)，使用 [Watson Studio 深度学习](https://www.ibm.com/cloud/deep-learning)服务来训练和维护模型

* 如果您刚开始学习，并且希望设置自己的 FfDL 部署，请遵照以下步骤。

## 步骤

1. [快速入门](#1-quick-start)
  - 1.1 [使用 Minikube 进行安装](#11-installation-using-minikube)
  - 1.2 [使用 Kubernetes 集群进行安装](#12-installation-using-kubernetes-cluster)
  - 1.3 [使用 IBM Cloud Kubernetes 集群进行安装](#13-installation-using-ibm-cloud-kubernetes-cluster)
2. [测试](#2-test)
3. [监视](#3-monitoring)
4. [开发](#4-development)
5. [详细的安装说明](#5-detailed-installation-instructions)
6. [详细的测试说明](#6-detailed-testing-instructions)
  - 6.1 [使用 FfDL 本地 S3 对象存储](#61-using-ffdl-local-s3-based-object-storage)
  - 6.2 [使用云对象存储](#62-using-cloud-object-storage)
7. [清理](#7-clean-up)
8. [故障排除](#8-troubleshooting)
9. [参考资料](#9-references)

## 1. 快速入门

可通过多种安装路径在本地（“一键式安装”）或现有的 Kubernetes 集群中安装 FfDL。

> 注意：如果您的 Kubernetes 集群版本是 1.7 或更低版本，请转至 [values.yaml](values.yaml)，并将 `k8s_1dot8_or_above` 更改为 **false**。

### 1.1 使用 Minikube 进行安装

如果您已在机器上安装了 Minikube，请使用以下命令来部署 FfDL 平台：
``` shell
export VM_TYPE=minikube
make minikube
make deploy
```

### 1.2 使用 Kubernetes 集群进行安装

要将 FfDL 安装到合适的 Kubernetes 集群中，请确保 `kubectl` 指向正确的名称空间，然后部署平台服务：
> 注意：对于 PUBLIC_IP，请记录一个可以访问集群的 NodePort 的集群公共 IP。

``` shell
export VM_TYPE=none
export PUBLIC_IP=<Cluster Public IP>
make deploy
```

### 1.3 使用 IBM Cloud Kubernetes 集群进行安装

要将 FfDL 安装到合适的 IBM Cloud Kubernetes 集群中，请确保 `kubectl` 指向正确的名称空间，并使用 `bx login` 登录到您的机器，然后部署平台服务：
``` shell
export VM_TYPE=ibmcloud
export CLUSTER_NAME=<Your Cluster Name> # Replace <Your Cluster Name> with your IBM Cloud Cluster Name
make deploy
```

## 2. 测试

提交本存储库中包含的训练作业简单示例（请参阅 `etc/examples` 文件夹）：

```
make test-submit
```

## 3. 监视

该平台随附简单的 Grafana 监视仪表板。在运行 `deploy` make target 时，会打印出 URL。

## 4. 开发

请参阅[开发人员指南](docs/developer-guide.md)，了解更多详细信息。

## 5. 详细的安装说明

1. 首先，克隆该存储库，并在 Kubernetes 集群上安装 Helm Tiller。
``` shell
helm init

# Make sure the tiller pod is Running before proceeding to the next step.
kubectl get pods --all-namespaces | grep tiller-deploy
# kube-system   tiller-deploy-fb8d7b69c-pcvc2              1/1       Running
```

2. 现在，我们使用 helm install 来安装所有必需的 FfDL 组件。
> 注意：如果您的 Kubernetes 集群版本是 1.7 或更低版本，请转至 [values.yaml](values.yaml)，并将 `k8s_1dot8_or_above` 更改为 **false**。

``` shell
helm install .
```
> 注意：如果您希望升级较早版本的 FfDL，请运行
> `helm upgrade $(helm list | grep ffdl | awk '{print $1}' | head -n 1) .`

在进行下一步之前，确保所有的 FfDL 组件都已安装并在运行。
``` shell
kubectl get pods
# NAME                                 READY     STATUS    RESTARTS   AGE
# alertmanager-7cf6b988b9-h9q6q        1/1       Running   0          5h
# etcd0                                1/1       Running   0          5h
# ffdl-lcm-65bc97bcfd-qqkfc            1/1       Running   0          5h
# ffdl-restapi-8777444f6-7jfcf         1/1       Running   0          5h
# ffdl-trainer-768d7d6b9-4k8ql         1/1       Running   0          5h
# ffdl-trainingdata-866c8f48f5-ng27z   1/1       Running   0          5h
# ffdl-ui-5bf86cc7f5-zsqv5             1/1       Running   0          5h
# mongo-0                              1/1       Running   0          5h
# prometheus-5f85fd7695-6dpt8          2/2       Running   0          5h
# pushgateway-7dd8f7c86d-gzr2g         2/2       Running   0          5h
# storage-0                            1/1       Running   0          5h

helm status $(helm list | grep ffdl | awk '{print $1}' | head -n 1) | grep STATUS:
# STATUS: DEPLOYED
```

3. 运行以下脚本，利用来自 prometheus 的日志记录信息配置 Grafana 以监控 FfDL。
> 注意：如果您正在使用 IBM Cloud 集群，请确保自己已通过 `bx login` 进行登录。

``` shell
# If your Cluster is running on Minikube, replace "ibmcloud" to "minikube"
# If your Cluster is not running on Minikube or IBM Cloud, replace "ibmcloud" to "none"
export VM_TYPE=ibmcloud

# Replace <Your Cluster Name> with your IBM Cloud Cluster Name if your cluster is on IBM Cloud.
# Use export PUBLIC_IP if you are using a none VM_TYPE. A Cluster Public IP that can access your Cluster's NodePorts.
export CLUSTER_NAME=<Your Cluster Name>
export PUBLIC_IP=<Cluster Public IP>

./bin/grafana.init.sh
```

4. 最后，运行以下命令来获取 Grafana、FfDL Web UI 和 FfDL REST API 端点。
``` shell
# Note: $(make --no-print-directory kubernetes-ip) simply gets the Public IP for your cluster.
node_ip=$(make --no-print-directory kubernetes-ip)

# Obtain all the necessary NodePorts for Grafana, Web UI, and RestAPI.
grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
ui_port=$(kubectl get service ffdl-ui -o jsonpath='{.spec.ports[0].nodePort}')
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')

# Echo statements to print out Grafana and Web UI URLs.
echo "Monitoring dashboard: http://$node_ip:$grafana_port/ (login: admin/admin)"
echo "Web UI: http://$node_ip:$ui_port/#/login?endpoint=$node_ip:$restapi_port&username=test-user"
```

祝贺您，FfDL 现在正在您的集群上运行。现在，您可以前往[第 6 步](#6-detailed-testing-instructions)运行一些样本作业，或者前往[用户指南](docs/user-guide.md)了解如何运行和部署自定义模型。

## 6. 详细的测试说明

在本示例中，我们将运行一些简单的作业，使用 TensorFlow 和 Caffe 来训练卷积网络模型。我们将下载一系列的 MNIST 手写体数字图像，通过对象存储来存储这些图像，然后使用 FfDL CLI 训练手写体数字分类模型。

> 注意：对于 Minikube，请确保您通过运行 `docker pull tensorflow/tensorflow` 获得了最新的 TensorFlow Docker 镜像

### 6.1. 使用 FfDL 本地 S3 对象存储

1. 运行以下命令，从集群中获取对象存储端点。
```shell
node_ip=$(make --no-print-directory kubernetes-ip)
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
s3_url=http://$node_ip:$s3_port
```

2. 下一步，设置缺省的对象存储访问 ID 和 KEY。然后，创建可容纳所有必需训练数据和训练模型的存储桶。
```shell
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;

s3cmd="aws --endpoint-url=$s3_url s3"
$s3cmd mb s3://tf_training_data
$s3cmd mb s3://tf_trained_model
$s3cmd mb s3://mnist_lmdb_data
$s3cmd mb s3://dlaas-trained-models
```

3. 现在，创建一个临时存储库，下载用于训练和标记 TensorFlow 模型所需的图像，然后将这些图像上传到 tf_training_data 存储桶。

```shell
mkdir tmp
for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz;
do
  test -e tmp/$file || wget -q -O tmp/$file http://yann.lecun.com/exdb/mnist/$file
  $s3cmd cp tmp/$file s3://tf_training_data/$file
done
```

4. 现在，您应当在对象存储中包含了所有必需的训练数据集。我们继续为深度学习即服务设置 REST API 端点和缺省凭证。完成之后，您就可以使用 FfDL CLI（可执行的二进制文件）开始运行作业。

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default TensorFlow model
CLI_CMD=$(pwd)/cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

祝贺您，您已在 FfDL 上提交了第一个作业。从 FfDL UI 或仅需运行 `$CLI_CMD list` 即可检查 FfDL 状态。

> 您可以通过[用户指南](docs/user-guide.md#2-create-new-models-with-ffdl)，学习如何创建自己的模型定义文件和 `manifest.yaml`。

5. 如果您希望通过 FfDL UI 运行作业，只需运行以下命令来创建自己的模型 zip 文件。

```shell
# Replace tf-model with the model you want to zip
pushd etc/examples/tf-model && zip ../tf-model.zip * && popd
```

然后，在 `etc/examples/` 存储库中上传 `tf-model.zip` 和  `manifest.yml`（缺省的 TensorFlow 模型），如下所示。
接着，单击 `Submit Training Job` 运行作业。

![ui-example](docs/images/ui-example.png)

6. （可选）使用 FfDL 上不同的深度学习框架来提交作业非常简单，让我们来尝试运行 Caffe 作业。为 Caffe 模型下载 [LMDB 格式](https://en.wikipedia.org/wiki/Lightning_Memory-Mapped_Database)的所有必需训练图像和测试图像，然后将这些图像上传到 mnist_lmdb_data 存储桶。

```shell
for phase in train test;
do
  for file in data.mdb lock.mdb;
  do
    tmpfile=tmp/$phase.$file
    test -e $tmpfile || wget -q -O $tmpfile https://github.com/albarji/caffe-demos/raw/master/mnist/mnist_"$phase"_lmdb/$file
    $s3cmd cp $tmpfile s3://mnist_lmdb_data/$phase/$file
  done
done
```

7. 现在开始训练 Caffe 作业。

```shell
$CLI_CMD train etc/examples/caffe-model/manifest.yml etc/examples/caffe-model
```

祝贺您，现在您已掌握如何通过不同的深度学习框架来部署作业。要了解有关作业执行结果的更多信息，只需运行 `$CLI_CMD logs <MODEL_ID>`

> 如果不再需要我们在本例中使用的任何 MNIST 数据集，只需删除 `tmp` 存储库。

### 6.2. 使用云对象存储

在本部分中，我们将演示如何通过云对象存储中存储的训练数据运行 TensorFlow 作业。

> 注释：这也可以通过其他云提供商的对象存储来完成，但是在本操作说明中，我们将演示如何使用 IBM Cloud Object Storage。

1. 配置您的云提供商提供的 S3 对象存储。记录认证端点、访问键 ID 以及密钥。

> 对于 IBM Cloud，您可以从 [IBM Cloud 仪表板](https://console.bluemix.net/catalog/infrastructure/cloud-object-storage?taxonomyNavigation=apps)或从 [SoftLayer 门户网站](https://control.softlayer.com/storage/objectstorage)配置对象存储。

2. 使用刚刚获得的对象存储凭证设置 S3 命令。

```shell
s3_url=http://<Your object storage Authentication Endpoints>
export AWS_ACCESS_KEY_ID=<Your object storage Access Key ID>
export AWS_SECRET_ACCESS_KEY=<Your object storage Access Key Secret>

s3cmd="aws --endpoint-url=$s3_url s3"
```

3. 下一步，我们来创建两个存储桶，一个用于存储训练数据，另一个用于存储训练结果。
```shell
trainingDataBucket=<unique bucket name for training data storage>
trainingResultBucket=<unique bucket name for training result storage>

$s3cmd mb s3://$trainingDataBucket
$s3cmd mb s3://$trainingResultBucket
```

4. 现在，创建一个临时存储库，下载用于训练和标记 TensorFlow 模型所需的图像，然后将这些图像上传到训练数据存储桶。

```shell
mkdir tmp
for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz;
do
  test -e tmp/$file || wget -q -O tmp/$file http://yann.lecun.com/exdb/mnist/$file
  $s3cmd cp tmp/$file s3://$trainingDataBucket/$file
done
```

5. 接下来，我们需要利用以下 sed 命令，修改示例作业以使用云对象存储。
```shell
if [ "$(uname)" = "Darwin" ]; then
  sed -i '' s#"tf_training_data"#"$trainingDataBucket"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"tf_trained_model"#"$trainingResultBucket"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"http://s3.default.svc.cluster.local"#"$s3_url"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"user_name: test"#"user_name: $AWS_ACCESS_KEY_ID"# etc/examples/tf-model/manifest.yml
  sed -i '' s#"password: test"#"password: $AWS_SECRET_ACCESS_KEY"# etc/examples/tf-model/manifest.yml
else
  sed -i s#"tf_training_data"#"$trainingDataBucket"# etc/examples/tf-model/manifest.yml
  sed -i s#"tf_trained_model"#"$trainingResultBucket"# etc/examples/tf-model/manifest.yml
  sed -i s#"http://s3.default.svc.cluster.local"#"$s3_url"# etc/examples/tf-model/manifest.yml
  sed -i s#"user_name: test"#"user_name: $AWS_ACCESS_KEY_ID"# etc/examples/tf-model/manifest.yml
  sed -i s#"password: test"#"password: $AWS_SECRET_ACCESS_KEY"# etc/examples/tf-model/manifest.yml
fi
```

6. 现在，您应当在训练数据存储桶中包含了所有必需的训练数据集。我们继续为深度学习即服务设置 REST API 端点和缺省凭证。完成之后，您就可以使用 FfDL CLI（可执行的二进制文件）开始运行作业。

```shell
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
export DLAAS_URL=http://$node_ip:$restapi_port; export DLAAS_USERNAME=test-user; export DLAAS_PASSWORD=test;

# Obtain the correct CLI for your machine and run the training job with our default TensorFlow model
CLI_CMD=cli/bin/ffdl-$(if [ "$(uname)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model
```

## 7. 清理
如果您希望从集群中移除 FfDL，只需使用以下命令或运行 `helm delete <your FfDL release name>`
```shell
helm delete $(helm list | grep ffdl | awk '{print $1}' | head -n 1)
```

## 8. 故障排除

* FfDL 仅在 Mac OS 和 Linux 下经过测试

* Mac OS 中 Minikube 的缺省驱动程序是 VirtualBox，但众所周知，该驱动程序在网络方面存在问题。我们通常建议 Mac OS 用户使用 xhyve 驱动程序来安装 Minikube。

* 另外，在本地测试 Minikube 时，请确保将 `docker` CLI 指向 Minikube 的 Docker 守护程序：

   ```
   eval $(minikube docker-env)
   ```
* 如果您使用 Minikube 时遇到 DNS 名称解析问题，那么确保系统仅使用 `10.0.0.10` 作为单一名称服务器。使用多个名称服务器会导致各种问题，尤其是在 Mac OS 下。

* 如果 `glide install` 失败，出现表示路径不存在的错误（例如，“Without src, cannot continue”），请确保遵循标准的 Go 目录布局（请参见 [前提条件部分]{#Prerequisites}）。

* 要移除集群上的 FfDL，只需运行 `make undeploy`

* 在使用 FfDL CLI 来训练模型时，请确保目录路径末尾不含反斜杠 `/`。

## 9. 参考资料

根据 IBM 研究院在深度学习方面的工作成果

* B. Bhattacharjee 等人，“IBM Deep Learning Service”，IBM Journal of Research and Development，第 61 卷，第 4 号，第 10:1-10:11 页，2017 年 7 月 - 9 月 1 日。https://arxiv.org/abs/1709.05871

* Scott Boag 等人，Scalable Multi-Framework Multi-Tenant Lifecycle Management of Deep Learning Training Jobs，NIPS'17 会议上有关 ML 系统的研讨会成果，2017 年。http://learningsys.org/nips17/assets/papers/paper_29.pdf
