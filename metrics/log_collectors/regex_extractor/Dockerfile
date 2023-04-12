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

FROM ubuntu:16.04

# TODO: Just use community Python 3.6 image per conversation with Atin.

RUN apt-get update && apt-get install -y --no-install-recommends software-properties-common=0.96.*

RUN add-apt-repository ppa:deadsnakes/ppa
# runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
		tcl=8.6.* \
		tk=8.6.* \
		vim=2:7.4.* \
		mlocate=0.26-* \
		wget=1.17.* \
		gcc=4:5.3.* \
	&& rm -rf /var/lib/apt/lists/*

ENV PYTHON_VERSION 3.6.3

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc=4:5.3.* \
    make=4.1-* \
    musl-dev=1.1.* \
    "python3.6" \
    python3-pip=8.1.* \
	cron=3.0pl1-*

RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.5 1
RUN update-alternatives --install /usr/bin/python3 python3 /usr/bin/python3.6 3

RUN update-alternatives --config python3

RUN pip3 install --upgrade pip
RUN pip3 install -U setuptools
RUN pip3 install grpcio==1.7.3 python-dateutil pyyaml
RUN pip3 install awscli

# make some useful symlinks that are expected to exist
RUN cd /usr/local/bin \
	&& ln -s idle3 idle \
	&& ln -s pydoc3 pydoc \
	&& ln -s python3 python \
	&& ln -s python3-config python-config

RUN mkdir -p /scripts
RUN mkdir -p /scripts/patterns
RUN mkdir -p /scripts/log_collectors
RUN mkdir -p /scripts/log_collectors/training_data_service_client
RUN mkdir -p /scripts/log_collectors/training_data_service_client/certs

ADD src/*.* /scripts/
ADD src/patterns/* /scripts/patterns/

ADD training_data_service_client/*.* /scripts/log_collectors/training_data_service_client/
ADD training_data_service_client/certs/*.* /scripts/log_collectors/training_data_service_client/certs/

RUN mkdir -p /scripts/log_collectors/training_data_service_client/certs

WORKDIR /scripts

RUN echo python=python3 > /root/.bashrc