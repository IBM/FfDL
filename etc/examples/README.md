# List of FfDL Examples

* `tf-model`: Sample TensorFlow training job example that is using 2 convolutional layers with MNIST.

* `tf-summary-model`: Sample TensorFlow training job that demonstrates how to store emetrics to TensorBoard on FfDL.

* `tf-distributed`: Sample TensorFlow job with native distributed training using parameter server.

* `pytorch-model`: Sample PyTorch training job with MNIST.

* `keras-test`: Sample Keras training job with TensorFlow backend.

* `caffe-model`: Sample Caffe training job using MNIST.

* `horovod`: Sample TensorFlow and PyTorch training job with [Horovod's](https://github.com/uber/horovod) MPI distributed training approach.

* `pytorch-launch-dist`: Sample PyTorch distributed training job activated with PyTorch's native launch function. (PyTorch 0.4.1)

* `pytorch-dist-onnx`: PyTorch native distributed training with model exported in ONNX format. (Fashion MNIST model using 2 Convolutional Layers) (PyTorch 0.4.1)

* `c10d-native-parallelism`: PyTorch 1.0 Distributed Data Parallelism example with NCCL communication backend. This example will assign only one rank per node and demonstrate how to apply PyTorch's native Distributed Data Parallelism with multiple GPUs per node. (SEP 25 build)

* `c10d-onnx-mpi`: PyTorch 1.0 Distributed example with MPI communication backend and exported in ONNX. (SEP 25 build)

* `c10d-dist-onnx`: (Experimental) This is an updated version of `pytorch-dist-onnx` example with PyTorch 1.0 native distributed training using GLOO communication backend with model exported in ONNX format. Will merge with `pytorch-dist-onnx` once PyTorch 1.0 is released. (SEP 25 build)

* `c10d-mpi-parallelism`: (Experimental since PyTorch didn't claim MPI is working on the native Distributed Data Parallelism function) PyTorch 1.0 Distributed Data Parallelism function example with MPI communication backend. (SEP 25 build)
