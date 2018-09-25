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
import copy

from math import ceil
from random import Random
from torch.multiprocessing import Process
from torch.autograd import Variable
from torchvision import datasets, transforms


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


def get_dataset():
    """ Get FashionMNIST dataset """
    data_path = os.environ.get("DATA_DIR") + '/data'
    trainset = datasets.FashionMNIST(
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
    return trainset, testset


def average_gradients(model):
    """ Gradient averaging. """
    size = float(dist.get_world_size())
    for param in model.parameters():
        dist.all_reduce(param.grad.data, op=dist.reduce_op.SUM, group=dist.group.WORLD)
        param.grad.data /= size

# def multigpu_average_gradients(model):
#     """ Gradient averaging. """
#     size = float(dist.get_world_size())
#     tensor_list = []
#     for dev_idx in range(torch.cuda.device_count()):
#         tensor_list.append(torch.FloatTensor([1]).cuda(dev_idx))
#     dist.all_reduce_multigpu(tensor_list, op=dist.reduce_op.SUM, group=dist.group.WORLD)
#     for tensor in tensor_list:
#         tensor /= size*len(tensor_list)


def run(local_device, rank, size, batch_size, is_gpu, is_distributed):
    """ Distributed Synchronous SGD Example """
    torch.manual_seed(1234)
    train_set, test_set = get_dataset()
    result_dir = os.environ.get("RESULT_DIR") + '/saved_model'
    # For GPU use
    if is_gpu:
        #torch.cuda.set_device(local_device)
        model = Net().cuda()
    else:
        model = Net()
    if is_distributed:
        model = torch.nn.parallel.DistributedDataParallel(model)
        train_sampler = torch.utils.data.distributed.DistributedSampler(train_set,
            num_replicas=dist.get_world_size() , rank=dist.get_rank())
    train_set = torch.utils.data.DataLoader(
        train_set, batch_size=batch_size, shuffle=(train_sampler is None), sampler=train_sampler,
        pin_memory=True)
    test_set = torch.utils.data.DataLoader(
        test_set, batch_size=batch_size, shuffle=True, pin_memory=True)
    optimizer = optim.SGD(model.parameters(), lr=0.01, momentum=0.9)

    # To train model
    model.train()
    for epoch in range(100):
        epoch_loss = 0.0
        if is_distributed:
            train_sampler.set_epoch(epoch)
        for data, target in train_set:
            # For GPU use
            if is_gpu:
                data, target = data.cuda(), target.cuda()
            else:
                data, target = Variable(data), Variable(target)
            optimizer.zero_grad()
            output = model(data)
            loss = F.nll_loss(output, target)
            epoch_loss += loss.item()
            loss.backward()
            # NOTE: Scatter method was used in DistributedDataParallel
            # if not (size == 1):
            #     average_gradients(model)
            optimizer.step()
        print('Process ', rank,
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
                    data, target = data.cuda(), target.cuda()
                else:
                    data, target = Variable(data), Variable(target)
                output = model(data)
                test_loss += F.nll_loss(output, target, reduction="sum").item()
                pred = output.data.max(1, keepdim=True)[1]
                correct += pred.eq(target.data.view_as(pred)).sum().item()
            print('Test_set:  avg_loss: ', test_loss / len(test_set.dataset),
                  ', accuracy: ', 100. * correct / len(test_set.dataset), '%')

    # Save model
    if int(os.environ.get("LEARNER_ID")) == 1:
        torch.save(model.state_dict(), result_dir)
        # NOTE: ONNX doesn't support scatter operation yet.
        # dummy_input = ""
        # if is_gpu:
        #     dummy_input = Variable(torch.randn(1, 1, 28, 28)).cuda()
        # else:
        #     dummy_input = Variable(torch.randn(1, 1, 28, 28))
        # model_path = os.environ.get("RESULT_DIR") + "/pytorch-dist.onnx"
        # torch.onnx.export(model, dummy_input, model_path)


# Change 'backend' to appropriate backend identifier
def init_processes(local_device, rank, size, fn, path_to_file, batch_size, is_gpu, backend):
    """ Initialize the distributed environment. """
    print("Process ", rank, " connected")
    dist.init_process_group(backend,
                            init_method=path_to_file,
                            world_size=size, group_name="train_dist",
                            rank=rank)
    print("FOUND SHARED FILE")
    fn(local_device, rank, size, batch_size, is_gpu, True)

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
        run(0, 1, batch_size, (num_gpus == 1), False)
        print("COMPLETION TIME: ", time.time() - start_time)
    else:
        if num_gpus == 0:
            p = local_process(init_processes, (0, int(os.environ.get("LEARNER_ID")) - 1, world_size, run, data_dir, batch_size, False, 'gloo'))
            p.start()
            processes.append(p)
        else:
            for process_num in range(0, num_gpus):
                p = local_process(init_processes, (process_num, (process_num*int(os.environ.get("NUM_LEARNERS")) + int(os.environ.get("LEARNER_ID")) - 1, world_size, run, data_dir, batch_size, True, 'nccl'))
                p.start()
                processes.append(p)

        print("COMPLETION TIME: ", time.time() - start_time)

        # FfDL assume only the master learner job will terminate and store all
        # the logging file.
        if int(os.environ.get("LEARNER_ID")) != 1:
            while True:
                time.sleep(1000000)
