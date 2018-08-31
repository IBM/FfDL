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
import torch.onnx

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
    data_path = os.environ.get("DATA_DIR") + '/data'
    dataset = datasets.FashionMNIST(
        data_path,
        train=True,
        download=False,
        transform=transforms.Compose([
            transforms.ToTensor(),
            transforms.Normalize((0.1307, ), (0.3081, ))
        ]))
    testset = datasets.FashionMNIST(
        data_path,
        train=False,
        download=False,
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
    partition_testset = DataPartitioner(testset, partition_sizes)
    if is_distributed:
        partition = partition.use(dist.get_rank())
        partition_testset = partition_testset.use(dist.get_rank())
    else:
        partition = partition.use(0)
        partition_testset = partition_testset.use(0)
    train_set = torch.utils.data.DataLoader(
        partition, batch_size=bsz, shuffle=True)
    test_set = torch.utils.data.DataLoader(
        testset, batch_size=batch_size, shuffle=True)
    return train_set, test_set, bsz


def average_gradients(model):
    """ Gradient averaging. """
    size = float(dist.get_world_size())
    for param in model.parameters():
        dist.all_reduce(param.grad.data, op=dist.reduce_op.SUM, group=0)
        param.grad.data /= size


def run(rank, size, batch_size, is_gpu):
    """ Distributed Synchronous SGD Example """
    torch.manual_seed(1234)
    train_set, test_set, bsz = partition_dataset(batch_size, (not (size == 1)))
    result_dir = os.environ.get("RESULT_DIR") + '/saved_model'
    # For GPU use
    if is_gpu:
        device = torch.device("cuda:{}".format(rank))
        model = Net().to(device)
    else:
        model = Net()
        model = model
#    model = model.cuda(rank)
    optimizer = optim.SGD(model.parameters(), lr=0.01, momentum=0.9)

    num_batches = ceil(len(train_set.dataset) / float(bsz))
    # To train model
    model.train()
    for epoch in range(100):
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
        print('Process ', os.environ.get("LEARNER_ID"),
              ', epoch ', epoch, '. avg_loss: ',
              epoch_loss / len(train_set))

    # Test model
    if int(os.environ.get("LEARNER_ID")) == 1:
        model.eval()
        test_loss = 0.0
        correct = 0
        with torch.no_grad():
            for data, target in test_set:
                # For GPU use
                if is_gpu:
                    data, target = data.to(device), target.to(device)
                else:
                    data, target = Variable(data), Variable(target)
                output = model(data)
                test_loss += F.nll_loss(output, target, size_average=False).item()
                pred = output.data.max(1, keepdim=True)[1]
                correct += pred.eq(target.data.view_as(pred)).sum().item()
            print('Test_set:  avg_loss: ', test_loss / len(test_set.dataset),
                  ', accuracy: ', 100. * correct / len(test_set.dataset), '%')

    # Save model
    if int(os.environ.get("LEARNER_ID")) == 1:
        torch.save(model, result_dir)
        dummy_input = ""
        if is_gpu:
            dummy_input = Variable(torch.randn(1, 1, 28, 28)).cuda()
        else:
            dummy_input = Variable(torch.randn(1, 1, 28, 28))
        model_path = os.environ.get("RESULT_DIR") + "/pytorch-dist.onnx"
        torch.onnx.export(model, dummy_input, model_path)


# Change 'backend' to appropriate backend identifier
def init_processes(rank, size, fn, path_to_file, batch_size, is_gpu, backend):
    """ Initialize the distributed environment. """
    print("Process " + os.environ.get("LEARNER_ID") + " connected")
    dist.init_process_group(backend,
                            init_method=path_to_file,
                            world_size=size, group_name="train_dist")
    print("FOUND SHARED FILE")
    fn(rank, size, batch_size, is_gpu)

def local_process(target, args):
    return Process(target=target, args=args)

if __name__ == "__main__":

    parser = argparse.ArgumentParser()
    parser.add_argument('--batch_size', help='Specify the batch size to be used in training')
    args = parser.parse_args()

    batch_size = args.batch_size
    # Default batch size is set to 1024. When using a large numbers of learners,
    # a larger batch size is sometimes necessary to see speed improvements.
    if batch_size is None:
        batch_size = 1024
    else:
        batch_size = int(batch_size)

    start_time = time.time()
    num_gpus = int(float(os.environ.get("GPU_COUNT")))
    if num_gpus == 0:
        world_size = int(os.environ.get("NUM_LEARNERS"))
    else:
        world_size = num_gpus * int(os.environ.get("NUM_LEARNERS"))
    data_dir = "file:///job/" + os.environ.get("TRAINING_ID")
    processes = []
    print("data_dir is " + data_dir)

    if world_size == 1:
        run(0, 1, batch_size, (num_gpus == 1))
        print("COMPLETION TIME: ", time.time() - start_time)
    else:
        if num_gpus == 0:
            p = local_process(init_processes, (0 , world_size, run, data_dir, batch_size, False, 'gloo'))
            p.start()
            processes.append(p)

        for process_num in range(0, num_gpus):
            p = local_process(init_processes, (process_num, world_size, run, data_dir, batch_size, True, 'gloo'))
            p.start()
            processes.append(p)

        for p in processes:
            p.join()

        print("COMPLETION TIME: ", time.time() - start_time)

        # FfDL assume only the master learner job will terminate and store all
        # the logging file.
        if int(os.environ.get("LEARNER_ID")) != 1:
            while True:
                time.sleep(1000000)
