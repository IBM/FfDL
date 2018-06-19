#
# Copyright 2017-2018 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import torch
import numpy as np
from torchvision import datasets, transforms
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
from torch.autograd import Variable
import boto3
import botocore
import tarfile
import os

class Net(nn.Module):
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
        return F.log_softmax(x)


class PyMnist(object):
    def __init__(self):
        training_id = os.environ.get("TRAINING_ID")
        endpoint_url = os.environ.get("BUCKET_ENDPOINT_URL")
        bucket_name = os.environ.get("BUCKET_NAME")
        bucket_key = os.environ.get("BUCKET_KEY")
        bucket_secret = os.environ.get("BUCKET_SECRET")
        print("Training id:{} endpoint URL:{} key:{} secret:{}".format(training_id,endpoint_url,bucket_key,bucket_secret))
        
        self.class_names = ["class:{}".format(str(i)) for i in range(10)]
        #self.class_names = ["prediction"]

        # Define S3 resource and download the model file
        client = boto3.resource(
            's3',
            endpoint_url=endpoint_url,
            aws_access_key_id=bucket_key,
            aws_secret_access_key=bucket_secret,
        )

        KEY = training_id + '/saved_model.tar.gz' # replace with your object key

        try:
            client.Bucket(bucket_name).download_file(KEY, 'saved_model.tar.gz')
        except botocore.exceptions.ClientError as e:
            if e.response['Error']['Code'] == "404":
                print("The object does not exist.")
            else:
                raise

        # Untar model file
        tar = tarfile.open("saved_model.tar.gz")
        tar.extractall()
        tar.close()

        self.model = Net()
        self.model.load_state_dict(torch.load("./model.dat"))
        

    def predict(self,X,feature_names):
        tensor = torch.from_numpy(X).view(-1, 28, 28)
        t = transforms.Normalize((0.1307,), (0.3081,))
        tensor_norm = t(tensor)
        tensor_norm = tensor_norm.unsqueeze(0)
        out = self.model(tensor_norm.float())
        predictions = torch.nn.functional.softmax(out)
        print(predictions)
        return predictions.detach().numpy()


