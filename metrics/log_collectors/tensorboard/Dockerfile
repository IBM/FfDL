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


FROM tensorflow/tensorflow:1.5.0-py3
LABEL maintainer="Scott Boag <scott_boag@us.ibm.com>"

#install Keras
RUN pip3 install keras h5py tensor2tensor

RUN pip3 install --upgrade pip
RUN pip3 install -U setuptools
RUN pip3 install grpcio==1.7.3 python-dateutil pyyaml

RUN mkdir -p /scripts
RUN mkdir -p /scripts/log_collectors
RUN mkdir -p /scripts/log_collectors/training_data_service_client
RUN mkdir -p /scripts/log_collectors/training_data_service_client/certs

ADD src/*.* /scripts/
ADD training_data_service_client/*.* /scripts/log_collectors/training_data_service_client/
ADD training_data_service_client/certs/*.* /scripts/log_collectors/training_data_service_client/certs/

ENV PYTHONPATH /usr/local/lib/python3.5/dist-packages:/tensorflow:/usr/local/lib/python3.5/dist-packages/tensorboard:/tensorflow/third_party/py
ENV TENSORFLOWVERSION 1.3
ENV PYTHONVERSION 3.5

WORKDIR /scripts
