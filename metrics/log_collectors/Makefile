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

GRPC_GIT=git@github.ibm.com:deep-learning-platform/grpc-parameter-server.git

all: build push

build: build-simple_log_collector build-emetrics_file build-regex_extractor build-tensorboard-all

push: push-simple_log_collector push-emetrics_file push-regex_extractor push-tensorboard-all

build-simple_log_collector:
	$(MAKE) -C ./simple_log_collector build

build-emetrics_file:
	$(MAKE) -C ./emetrics_file build

build-regex_extractor:
	$(MAKE) -C ./regex_extractor build

push-simple_log_collector:
	$(MAKE) -C ./simple_log_collector push

push-emetrics_file:
	$(MAKE) -C ./emetrics_file push

push-regex_extractor:
	$(MAKE) -C ./regex_extractor push

build-tensorboard-all:
	$(MAKE) -C ./tensorboard build

push-tensorboard-all:
	$(MAKE) -C ./tensorboard push

imagename:
	$(MAKE) -C ./simple_log_collector imagename
	$(MAKE) -C ./emetrics_file imagename
	$(MAKE) -C ./regex_extractor imagename
	$(MAKE) -C ./tensorboard imagename
