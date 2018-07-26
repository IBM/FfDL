# PyTorch MNIST distributed training example

For more details on PyTorch distributed package, please refer to https://pytorch.org/tutorials/intermediate/dist_tuto.html

#### Create working processes and using native pytorch method **dist.init_process_group** to sync and join all the worker nodes.
- Tested with Syncing using **Shared File System** where worker processes will share their location/group name information on a shared file. (currently using it on shared pvc among the ffdl learners)
- Tested with both CPU and GPU with **Gloo** Backend.

1. Run the below block of code to pre-process your data, then store them in your object storage bucket.
```python
from torchvision import datasets, transforms
dataset = datasets.MNIST(
    ".",
    train=True,
    download=True,
    transform=transforms.Compose([
        transforms.ToTensor(),
        transforms.Normalize((0.1307, ), (0.3081, ))
    ]))
```

2. Put the model files in zip format and submit the training job on FfDL.
