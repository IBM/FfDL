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


REPO?=docker.io/ffdl
IMAGE_NAME=dlaas-service-base
IMAGE_TAG_ALPINE?=alpine3.7
IMAGE_TAG_UBUNTU?=ubuntu16.04

TARGET ?= $(shell uname -m)
ARCHS ?= x86_64 ppc64le

IMAGE_NAME_TARGET = -$(TARGET)

SERVICE_BASE_IMAGE_NAME = ${IMAGE_NAME}${IMAGE_NAME_TARGET}

all: build push

build-deps:
	cd build/grpc-health-checker && make install-deps build-x86-64

build: build-deps build-ubuntu build-alpine

push: push-alpine push-ubuntu

build-alpine:
	docker build -q -f Dockerfile.alpine --build-arg ARCH=$(TARGET) -t ${IMAGE_NAME}:${IMAGE_TAG_ALPINE} .
	docker tag ${IMAGE_NAME}:${IMAGE_TAG_ALPINE} ${REPO}/${SERVICE_BASE_IMAGE_NAME}:${IMAGE_TAG_ALPINE}

build-ubuntu:
	docker build -q -f Dockerfile.ubuntu --build-arg ARCH=$(TARGET) -t ${IMAGE_NAME}:${IMAGE_TAG_UBUNTU} .
	docker tag ${IMAGE_NAME}:${IMAGE_TAG_UBUNTU} ${REPO}/${SERVICE_BASE_IMAGE_NAME}:${IMAGE_TAG_UBUNTU}

push-alpine:
	docker push ${REPO}/${SERVICE_BASE_IMAGE_NAME}:${IMAGE_TAG_ALPINE}

push-ubuntu:
	docker push ${REPO}/${SERVICE_BASE_IMAGE_NAME}:${IMAGE_TAG_UBUNTU}

.PHONY: build push clean
