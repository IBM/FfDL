# Sample Models

We will keep this as a running list of models being trained using Deep Learning as a Service within Watson Studio and 
FfDL, as long as we can point to external open source references.

To run these experiments using Fabric for Deep Learning (FFDL), you can simply clone the FfDL repository and follow the 
instructions over [here](https://github.com/IBM/FfDL/blob/master/etc/converter/ffdl-wml.md) to convert your 
training-runs.yml into FfDL's specification.

## Model Asset Exchange Models


### 1. IBM Code Model Asset Exchange

#### Adversarial Cryptography Experiment

This repository contains code to run an Adversarial-Crypto experiment on IBM Watson Machine Learning. This experiment 
performs adversarial training to learn trivial encryption functions. The experiment is based on the 2016 paper "Learning 
to Protect Communications with Adversarial Neural Cryptography" by Abadi and Andersen.

[MAX - Adversarial Cryptography Experiment](https://github.com/IBM/MAX-Adversarial-Cryptography) 

#### Spatial Transformer Network

This repository contains code to train and score a Spatial Transformer Network/

[MAX-Spatial Transformer Network](https://github.com/IBM/MAX-Spatial-Transformer-Network)


### 2. CIFAR10 with PyTorch 

The code here is forked from [kuangliu work with PyTorch on the CIFAR10 dataset](https://github.com/kuangliu/pytorch-cifar), 
and adapted for submitting the model to Watson Studio for training. It is meant to get you quick-started. 

[CIFAR10 with PyTorch](https://github.com/IBM/pytorch-cifar10-in-ibm-cloud)

### 3. Adversarial Robustness Toolbox (ART v0.1)

This is a library dedicated to adversarial machine learning. Its purpose is to allow rapid crafting and analysis of 
attacks and defense methods for machine learning models. The Adversarial Robustness Toolbox provides an implementation 
for many state-of-the-art methods for attacking and defending classifiers. 
You can use our [ART Model Robustness Check.ipynb](/etc/notebooks/art/) Jupyter notebook to experiment with ART 
on FfDL.

[Adversarial Robustness Toolbox (ART v0.1)](https://github.com/IBM/adversarial-robustness-toolbox)
