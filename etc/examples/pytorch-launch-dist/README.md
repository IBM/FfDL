# PyTorch MNIST distributed training with launch example (experimental)

For more details on PyTorch distributed package with launch, please refer to https://github.com/pytorch/pytorch/blob/master/torch/distributed/launch.py

## Steps

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
