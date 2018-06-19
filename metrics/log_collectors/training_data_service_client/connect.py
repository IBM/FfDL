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
import time

from log_collectors.training_data_service_client import training_data_pb2_grpc as td


def get_connection()->td.TrainingDataStub:
    with open('log_collectors/training_data_service_client/certs/server.crt') as f:
        certificate = f.read()

    credentials = grpc.ssl_channel_credentials(root_certificates=certificate)

    # TODO: Change these to be configurable when/if we get the viper issue straightened out.
    isTLSEnabled = True
    isLocal = False

    if isLocal:
        host_url = '127.0.0.1'
        port = '30015'
    else:
        training_data_namespace = os.environ["TRAINING_DATA_NAMESPACE"]
        host_url = "ffdl-trainingdata.%s.svc.cluster.local" % training_data_namespace
        port = '80'

    host_url = '{}:{}'.format(host_url, port)

    print("host_url: "+host_url)
    sys.stdout.flush()
    channel = None

    for retryCount in range(0, 10):
        try:
            if isTLSEnabled:
                channel = grpc.secure_channel(host_url, credentials,
                                              options=(('grpc.ssl_target_name_override', 'dlaas.ibm.com',),))
            else:
                channel = grpc.insecure_channel(host_url)

            if channel is not None:
                break

        except Exception as inst:
            print("Exception trying to connect:",
                  sys.exc_info()[0])
            print(inst)
            sys.stdout.flush()

        time.sleep(.5)

    if channel is not None:
        tdClient = td.TrainingDataStub(channel)
    else:
        tdClient = None

    return tdClient

