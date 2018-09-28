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

import onnx
import os
from ngraph_onnx.onnx_importer.importer import import_onnx_file
import ngraph as ng
import numpy as np

class ONNXMnist(object):

    def __init__(self):
        # Initialize variables
        training_id = os.environ.get("TRAINING_ID")
        model_file_name = os.environ.get("MODEL_FILE_NAME")
        mountPath = os.environ.get("MOUNT_PATH")

        # Replace with path of trained model
        model_path = mountPath + '/' + training_id + '/' + model_file_name

        # Load model from the FLEXVolume
        self.models = import_onnx_file(model_path)
        self.runtime = ng.runtime(backend_name='CPU')
        self.model = self.models[0]
        self.net = self.runtime.computation(self.model['output'], *self.model['inputs'])
        print("Model loaded")

    def predict(self,X,features_names):
        return self.net(X)
