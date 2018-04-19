# Sample Models

We will keep this as a running list of models which we are training on Deep Learning as a Service in Watson and FfDL, as long as there we can point to open source references.

To run these experiments using Fabric for Deep Learning (FFDL), you can simply clone the FfDL repository and follow the instructions over [here]https://github.com/IBM/FfDL/blob/master/etc/converter/ffdl-wml.md) to convert your training-runs.yml into FfDL's specification.

## Model Asset Exchange Models


### IBM Code Model Asset Exchange

#### Adversarial Cryptography Experiment

This repository contains code to run an Adversarial-Crypto experiment on IBM Watson Machine Learning. This experiment performs adversarial training to learn trivial encryption functions. The experiment is based on the 2016 paper "Learning to Protect Communications with Adversarial Neural Cryptography" by Abadi and Andersen.

[MAX - Adversarial Cryptography Experiment](https://github.com/IBM/MAX-Adversarial-Cryptography) 

#### spatial Transformer Network

This repository contains code to train and score a Spatial Transformer Network/

[MAX-Spatial Transformer Network](https://github.com/IBM/MAX-Spatial-Transformer-Network)


### CIFAR10 with PyTorch 

The code here is forked from [kuangliu work with PyTorch on the CIFAR10 dataset](https://github.com/kuangliu/pytorch-cifar), and adapted for submitting the model to Watson Studio for training. It is meant to get you quick-started. 

[CIFAR10 with PyTorch](https://github.com/IBM/pytorch-cifar10-in-ibm-cloud)

### Adversarial Robustness Toolbox (ART v0.1)

This is a library dedicated to adversarial machine learning. Its purpose is to allow rapid crafting and analysis of attacks and defense methods for machine learning models. The Adversarial Robustness Toolbox provides an implementation for many state-of-the-art methods for attacking and defending classifiers. We will be adding support here for running these experiments on FfDL

[Adversarial Robustness Toolbox (ART v0.1)](https://github.com/IBM/adversarial-robustness-toolbox)