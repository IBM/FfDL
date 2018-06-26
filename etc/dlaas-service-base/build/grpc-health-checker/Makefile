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


GRPC_DIR = /go/src/github.com/IBM/FfDL/grpc-health-checker

install-deps:
	glide -q install

build-local: clean
	go build -o bin/grpc-health-checker

build-x86-64: clean
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo -o bin/grpc-health-checker

clean:
	rm -rf bin/

build-in-docker:
	docker run -i --name grpc-health-checker --rm --volume $${PWD}:$(GRPC_DIR) -w $(GRPC_DIR) golang:latest sh -c "make install-deps build-x86-64 test"
