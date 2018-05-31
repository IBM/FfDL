# Deploy FfDL Trained Models with Seldon

[Seldon](https://github.com/SeldonIO/seldon-core) provides a deployment platform to allow machine learning models to be exposed via REST or gRPC endpoints. Runtime graphs of models, routers (e.g., AB-tests, Multi-Armed bandits) , transformers (e.g., Feature normalization) and combiners (e.g., ensemblers) can be described using a Custom Kubernetes Resource JSON/YAML and then deployed, scaled and managed.

Any FyDL model whose runtime inference can be packaged as a Docker container can be managed by Seldon.

# Install Seldon

To install Seldon on your Kubernetes cluster next to FfDL see [here](https://github.com/SeldonIO/seldon-core/blob/master/docs/install.md).

# Deployment Steps

To deploy your models on Seldon you need to

 1. Wrap your runtime inference components as Docker containers that follow the Seldon APIs
 1. Describe the runtime graph as a Custom Kubernetes ```SeldonDeployment``` Resource
 1. Apply the graph via the Kubernetes API, e.g. using kubectl

# Examples

[train and deploy a Tensorflow MNIST classififer.](./tf-model/README.md)

