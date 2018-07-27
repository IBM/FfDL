"""
Distributed Learning using Pytorch's torch.distributed.launcher
"""

import time
import argparse
import sys
import os
import threading
import torch
import torch.distributed as dist
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim

from math import ceil
from random import Random
from torch.multiprocessing import Process
from torch.autograd import Variable
from torchvision import datasets, transforms

class Partition(object):
    """ Dataset-like object, but only access a subset of it. """

    def __init__(self, data, index):
        self.data = data
        self.index = index

    def __len__(self):
        return len(self.index)

    def __getitem__(self, index):
        data_idx = self.index[index]
        return self.data[data_idx]


class DataPartitioner(object):
    """ Partitions a dataset into different chuncks. """

    def __init__(self, data, sizes=[0.7, 0.2, 0.1], seed=1234):
        self.data = data
        self.partitions = []
        rng = Random()
        rng.seed(seed)
        data_len = len(data)
        indexes = [x for x in range(0, data_len)]
        rng.shuffle(indexes)

        for frac in sizes:
            part_len = int(frac * data_len)
            self.partitions.append(indexes[0:part_len])
            indexes = indexes[part_len:]

    def use(self, partition):
        return Partition(self.data, self.partitions[partition])


class Net(nn.Module):
    """ Network architecture. """

    def __init__(self):
        super(Net, self).__init__()
        self.conv1 = nn.Conv2d(1, 10, kernel_size=5)
        self.conv2 = nn.Conv2d(10, 20, kernel_size=5)
        self.conv2_drop = nn.Dropout2d()
        self.fc1 = nn.Linear(320, 50)
        self.fc2 = nn.Linear(50, 10)

    def forward(self, x):
        x = F.relu(F.max_pool2d(self.conv1(x), 2))
        x = F.relu(F.max_pool2d(self.conv2_drop(self.conv2(x)), 2))
        x = x.view(-1, 320)
        x = F.relu(self.fc1(x))
        x = F.dropout(x, training=self.training)
        x = self.fc2(x)
        return F.log_softmax(x, dim=1)


def partition_dataset(batch_size, is_distributed):
    """ Partitioning MNIST """
    vision_data = os.environ.get("DATA_DIR") + "/data"
    dataset = datasets.MNIST(
        vision_data,
        train=True,
        download=True,
        transform=transforms.Compose([
            transforms.ToTensor(),
            transforms.Normalize((0.1307, ), (0.3081, ))
        ]))
    if is_distributed:
        size = dist.get_world_size()
    else:
        size = 1
    bsz = int(batch_size / float(size))
    partition_sizes = [1.0 / size for _ in range(size)]
    partition = DataPartitioner(dataset, partition_sizes)
    if is_distributed:
        partition = partition.use(dist.get_rank())
    else:
        partition = partition.use(0)
    train_set = torch.utils.data.DataLoader(
        partition, batch_size=bsz, shuffle=True)
    return train_set, bsz


def average_gradients(model):
    """ Gradient averaging. """
    size = float(dist.get_world_size())
    for param in model.parameters():
        dist.all_reduce(param.grad.data, op=dist.reduce_op.SUM, group=0)
        param.grad.data /= size


def run(rank, world_rank, batch_size, is_gpu):
    """ Distributed Synchronous SGD Example """
    torch.manual_seed(1234)
    size = os.environ.get("WORLD_SIZE")
    train_set, bsz = partition_dataset(batch_size, (not (size == 1)))
    # For GPU use
    if is_gpu:
        device = torch.device("cuda:{}".format(rank))
        model = Net().to(device)
    else:
        model = Net()
        model = model
#    model = model.cuda(rank)
    optimizer = optim.SGD(model.parameters(), lr=0.01, momentum=0.5)

    num_batches = ceil(len(train_set.dataset) / float(bsz))
    for epoch in range(10):
        epoch_loss = 0.0
        for data, target in train_set:
            # For GPU use
            if is_gpu:
                data, target = data.to(device), target.to(device)
            else:
                data, target = Variable(data), Variable(target)
#            data, target = Variable(data.cuda(rank)), Variable(target.cuda(rank))
            optimizer.zero_grad()
            output = model(data)
            loss = F.nll_loss(output, target)
            epoch_loss += loss.item()
            loss.backward()
            if not (size == 1):
                average_gradients(model)
            optimizer.step()
        print('Process ', world_rank,
              ', epoch ', epoch, ': ',
              epoch_loss / num_batches)

# Change 'backend' to appropriate backend identifier
def init_processes(local_rank, world_rank, fn, batch_size, is_gpu, backend):
    """ Initialize the distributed environment. """
    print("World Rank: " + str(world_rank) + "  Local Rank: " + str(local_rank)  + " connected")
    dist.init_process_group(backend, init_method="env://")
    print("GROUP CREATED")
    fn(local_rank, world_rank, batch_size, is_gpu)


if __name__ == "__main__":

    parser = argparse.ArgumentParser()
    parser.add_argument("--local_rank", type=int)
    parser.add_argument("--batch_size", type=int)
    args = parser.parse_args()
    local_rank = args.local_rank
    batch_size = args.batch_size

    world_rank = os.environ.get("RANK")

    backend = 'gloo'
    start_time = time.time()

    init_processes(local_rank, world_rank, run, batch_size, True, backend)
    print("COMPLETION TIME: " + str(time.time() - start_time))

    if int(os.environ.get("LEARNER_ID")) == 1:
        print("Destroying Process Group")
        torch.distributed.destroy_process_group()
    else:
        while True:
            time.sleep(1000000)
