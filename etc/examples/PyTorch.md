PyTorch is a key part of the IBM open source and product offerings, and both [Watson Studio Deep Learning](https://www.ibm.com/cloud/deep-learning) and IBM [PowerAI](https://developer.ibm.com/linuxonpower/deep-learning-powerai/) support it. [PowerAI Enterprise with Spectrum Conductor Deep Learning Impact (DLI)](https://www.ibm.com/developerworks/community/blogs/281605c9-7369-46dc-ad03-70d9ad377480/entry/Dynamic_and_Resilient_Elastic_Deep_Learning_with_IBM_Spectrum_Conductor_Deep_Learning_Impact?lang=en_us) will be adding PyTorch 1.0 support, and the PowerAI version of PyTorch includes support for the IBM Deep learning Distributed library (DDL) for performing training across a cluster

Additionally, IBM has contributors supporting the open source PyTorch codebase, and we are adding multi-architecture support in PyTorch by enabling builds for Power architecture. There are other interesting projects that came out of IBM Research like Large Model Support and an [open source framework for seq2seq models](https://github.com/IBM/pytorch-seq2seq) in PyTorch.

Fabric for Deep Learning has support for the distributed deep learning training capability found in PyTorch 1.0 to run with its latest distributed learning back end. FfDL can provision the requested number of nodes and GPUs with a shared file system on Kubernetes that lets each node easily initialize and synchronize with the collective process group. From there, users can update gradients with various point-to-point, collective, or multi-GPU collective communication. We also provide several examples to demonstrate how to get started with defining the PyTorch process group with different types of communication back ends, then train the model with distributed data parallelism.

We've also fully tested FfDL with the new PyTorch distributed training with GLOO, NCCL, and MPI communication back ends to sync the model parameters.

In addition, we also support [PyTorch 0.41 distributed training leveraging Uber's Horovod mechanism](https://developer.ibm.com/code/2018/07/18/scalable-distributed-training-using-horovod-in-ffdl/).

## Distributed training leveraging PyTorch 1.0

Fabric for Deep Learning (FfDL) now supports both PyTorch 1.0 and the ONNX model format.

![diagram](images/pytorch-ffdl-blog.png)


|     | GLOO | MPI | NCCL |
|-----|:----:|:---:|:----:|
| CPU |   &#10004;  |  &#10004;  |   x  |
| GPU |   &#10004;  |  &#10004;  |   &#10004;  |

## Tech Preview for ONNX

In addition, we also have a tech preview of ONNX, which is a key feature of PyTorch 1.0. 

To save the models in ONNX format, you can run your usual model training functions to train the model and save the model using the native torch.onnx function similar to saving a PyTorch model. This removed the abstractions between converting within the different training and serving frameworks you have in your organization. After you have your model converted to ONNX, you can simply load it to any serving back end and start using the model.

## Complete the pipeline: Deploy your ONNX-based models using Seldon with nGraph

And to complete the pipeline, Fabric for Deep Learning has integration with Seldon. Apart from serving PyTorch and TensorFlow models, Seldon recently announced the ability to serve ONNX models with an nGraph back end, designed to optimize the inferencing performance, using CPUs.

With this, we can craft an end-to-end pipeline to convert FfDL-trained models to ONNX and serve it with Seldon. Furthermore, because FfDL can save trained models to Object Storage using the Flex volume on Kubernetes, we have improved the integration with Seldon as well to load the saved model directly from the FLEX volume, which can save the serving image disk space, generalize model wrapper definition, and improve scalability.

## Get started with PyTorch 1.0, ONNX, and FfDL today

FfDL with PyTorch 1.0 support is now available on GitHub, along with [AI Fairness 360](https://github.com/IBM/AIF360), [Adversarial Robustness Toolbox (ART)](https://github.com/IBM/adversarial-robustness-toolbox),  [Model Asset Exchange (MAX)](https://developer.ibm.com/code/exchanges/models/), and other open source AI projects from the [Center for Open Source Data and AI Technologies](https://developer.ibm.com/code/open/centers/codait/) group.

We hope you'll explore these tools and share your feedback. As with any open source project, its quality is only as good as the contributions it receives from the community.
