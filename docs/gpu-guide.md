# Deploy GPU Workloads on FfDL

***Deploy GPU jobs on FfDL still under development, any of the following instructions may have significant changes in the future.***

### Prerequisites

* You need to have [FfDL](../README.md#5-detailed-installation-instructions) running on your Cluster.

* You need to have a GPU cluster enabled with [feature gate `Accelerators`](https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/).

## Instructions

Once you have satisfy the prerequisites, you can following the same [testing instructions](../README.md#6-detailed-testing-instructions) on the main README and replace `$CLI_CMD train etc/examples/tf-model/manifest.yml etc/examples/tf-model` with `$CLI_CMD train etc/examples/tf-gpu-model/manifest.yml etc/examples/tf-gpu-model` to run the sample TensorFlow job on GPU.

You can go to the [user guide](user-guide.md) to learn more about how to modify the model manifest file and run GPU jobs with your own setting. Note that you must select the framework versions that support GPU and set the `gpus` section greater than 0 in order to execute your job with GPU.
