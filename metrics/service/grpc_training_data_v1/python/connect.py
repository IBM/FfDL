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

import sys
import os
import grpc

if sys.version_info[0] < 3:
    from training_data_service_client import training_data_pb2_grpc as td
else:
    from .training_data_pb2 import training_data_pb2_grpc as td

def get_connection():
    with open('./certs/server.crt') as f:
        certificate = f.read()

    credentials = grpc.ssl_channel_credentials(root_certificates=certificate)

    isTLSEnabled = True
    isLocal = False

    if isLocal:
        hosturl = '127.0.0.1'
        port = '30015'
    else:
        training_data_namespace = os.environ["TRAINING_DATA_NAMESPACE"]
        hosturl = "ffdl-trainingdata.%s.svc.cluster.local" % training_data_namespace
        port = '80'
        # hosturl = "10.177.1.186"
        # port = '80'

    hosturl = '{}:{}'.format(hosturl, port)

    print("hosturl: "+hosturl)
    sys.stdout.flush()

    if isTLSEnabled:
        channel = grpc.secure_channel(hosturl, credentials, options=(('grpc.ssl_target_name_override', 'dlaas.ibm.com',),))
    else:
        channel = grpc.insecure_channel(hosturl)

    tdClient = td.TrainingDataStub(channel)

    return tdClient

