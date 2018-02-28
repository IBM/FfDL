# Train Models using GPUs

***This is still under development, any of the following instructions may have significant changes in the future.***

### Prerequisites

* You need to have [FfDL](../README.md#5-detailed-installation-instructions) running on your Cluster.

* You need to have a Kubernetes cluster configured to use GPUs. Currently tested with Kubernetes configured using [feature gate `Accelerators`](https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/).

* Currently Tensorflow and Caffe tested with GPUs

## Instructions

### TensorFlow example

To run the TensorFlow job with GPU, simply go to the [tf-model's manifest file](../etc/examples/tf-model/manifest.yml) and do the following changes
* Change the framework version from `latest` to `latest-gpu` for CUDA 9 Driver or `1.3.0-gpu` for CUDA 8 Driver.
* Change the `gpus` section to be greater than 0, so the learner can get GPU resource to train the job.

The `etc/examples/tf-model/gpu-manifest.yml` is the example manifest file for running the TensorFlow example with GPU. Once you have done the above changes, you can following the same [testing instructions](../README.md#6-detailed-testing-instructions) on the main README to run the sample TensorFlow job on GPU.

### Caffe example

To run the Caffe job with GPU, simply go to the [caffe-model's manifest file](../etc/examples/caffe-model/manifest.yml) and do the following changes
* Change the Framework version from `cpu` to `gpu`.
* Change the `gpus` section to be greater than 0, so the learner can get GPU resource to train the job.
* Add the caffe GPU flag in the `command` section (e.g. Change the `command` from `caffe train -solver lenet_solver.prototxt` to `caffe train -gpu all -solver lenet_solver.prototxt`).
* Lastly, go to the `lenet_solver.prototxt` file and change `solver_mode` to GPU to enable Caffe to run on GPU.

The `etc/examples/caffe-model/gpu-manifest.yml` is the example manifest file for running the Caffe example with GPU. Once you have done the above changes, you can following the same [testing instructions](../README.md#6-detailed-testing-instructions) on the main README to run the sample TensorFlow job on GPU.

You can go to the [user guide](user-guide.md) to learn more about how to modify the model manifest file and run GPU jobs with your own setting. Note that you must select the framework versions that support GPU and set the `gpus` section greater than 0 in order to execute your job with GPU in the manifest file.
