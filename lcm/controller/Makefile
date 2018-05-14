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


REPO=docker.io/ffdl
CONTROLLER_IMAGE_NAME=controller
WHOAMI ?= $(shell whoami)
IMAGE_TAG ?= user-$(WHOAMI)

STATE_MACHINE_PDF=docs/state-machine.pdf

all: build push

build: clean
	docker build --label git-commit=$(shell git rev-list -1 HEAD) -t ${CONTROLLER_IMAGE_NAME}:${IMAGE_TAG} -f Dockerfile .
	docker tag ${CONTROLLER_IMAGE_NAME}:${IMAGE_TAG} ${REPO}/${CONTROLLER_IMAGE_NAME}:${IMAGE_TAG}

push:
	docker push ${REPO}/${CONTROLLER_IMAGE_NAME}:${IMAGE_TAG}

docs:
	cat docs/state-machine.dot |docker run -i --entrypoint /usr/bin/dot andrewshawcare/graphviz -T pdf > ${STATE_MACHINE_PDF}


clean:
	rm -rf build

.PHONY: build push docs clean

