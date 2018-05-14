TMPDIR ?= /tmp
UNAME = $(shell uname)
UNAME_SHORT = $(shell if [ "$(UNAME)" = "Darwin" ]; then echo 'osx'; else echo 'linux'; fi)
SERVICES = metrics lcm trainer restapi jobmonitor
IMAGES = metrics lcm trainer restapi jobmonitor controller databroker_objectstorage databroker_s3
THIS_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
DOCKER_REPO ?= docker.io
DOCKER_REPO_USER ?= user-test
DOCKER_REPO_PASS ?= test
DOCKER_REPO_DIR ?= ~/docker-registry/
DOCKER_NAMESPACE ?= ffdl
DOCKER_PULL_POLICY ?= IfNotPresent
DLAAS_LEARNER_REGISTRY ?= ${DOCKER_REPO}/${DOCKER_NAMESPACE}
INVENTORY ?= ansible/envs/local/minikube.ini
IMAGE_NAME_PREFIX = ffdl-
WHOAMI ?= $(shell whoami)
IMAGE_TAG ?= user-$(WHOAMI)
TEST_SAMPLE ?= tf-model
# VM_TYPE is "vagrant", "minikube" or "none"
VM_TYPE ?= minikube
HAS_STATIC_VOLUMES?=false
TEST_USER = test-user
SET_LOCAL_ROUTES ?= 0
MINIKUBE_RAM ?= 4096
MINIKUBE_CPUS ?= 3
MINIKUBE_DRIVER ?= hyperkit
MINIKUBE_BRIDGE ?= $(shell (ifconfig | grep -e "^bridge100:" || ifconfig | grep -e "^bridge0:") | sed 's/\(.*\):.*/\1/')
UI_REPO = git@github.com:IBM/FfDL-dashboard.git
CLI_CMD = $(shell pwd)/cli/bin/ffdl-$(UNAME_SHORT)
CLUSTER_NAME ?= mycluster
PUBLIC_IP ?= 127.0.0.1
CI_MINIKUBE_VERSION ?= v0.25.1
CI_KUBECTL_VERSION ?= v1.9.4

AWS_ACCESS_KEY_ID ?= test
AWS_SECRET_ACCESS_KEY ?= test

# This will used for "volume.beta.kubernetes.io/storage-class" for the shared volume
# Most likely "standard" in Minikube and "" in DIND, (other value is "default" or "ibmc-s3fs-standard")
# Use export SHARED_VOLUME_STORAGE_CLASS="ibmc-file-gold" for IBM Cloud deployment
ifeq ($(VM_TYPE),minikube)
SHARED_VOLUME_STORAGE_CLASS ?= "standard"
else
SHARED_VOLUME_STORAGE_CLASS ?= ""
endif

IMAGE_DIR := $(IMAGE_NAME)
ifneq ($(filter $(IMAGE_NAME),controller ),)
    IMAGE_DIR := lcm/$(IMAGE_NAME)
endif
ifeq ($(IMAGE_NAME),databroker_objectstorage)
    IMAGE_DIR := databroker/objectstorage
endif
ifeq ($(IMAGE_NAME),databroker_s3)
    IMAGE_DIR := databroker/s3
endif


# Main targets

usage:            ## Show this help
	@fgrep -h " ## " $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

build:            ## Build the code for all services
build: $(addprefix build-, $(SERVICES))

docker-build-images: $(addprefix docker-build-, $(IMAGES))

docker-build:     ## Build the Docker images for all services
docker-build: docker-build-images docker-build-ui docker-build-logcollectors

docker-push:
	@if [[ -z "${DOCKER_REPO_USER}" ]] || [[ -z "${DOCKER_REPO_PASS}" ]] ; then \
		echo "Please define DOCKER_REPO_USER and DOCKER_REPO_PASS."; \
		exit 1; \
	else \
		docker login --username=${DOCKER_REPO_USER} --password=${DOCKER_REPO_PASS} https://${DOCKER_REPO}; \
		docker tag ${DOCKER_NAMESPACE}/emetrics_file_extractor:${IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_NAMESPACE}/emetrics_file_extractor:${IMAGE_TAG} && docker rmi ${DOCKER_NAMESPACE}/emetrics_file_extractor:${IMAGE_TAG}; \
		docker tag ${DOCKER_NAMESPACE}/log_collector:${IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_NAMESPACE}/log_collector:${IMAGE_TAG} && docker rmi ${DOCKER_NAMESPACE}/log_collector:${IMAGE_TAG}; \
		docker tag ${DOCKER_NAMESPACE}/regex_extractor:${IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_NAMESPACE}/regex_extractor:${IMAGE_TAG} && docker rmi ${DOCKER_NAMESPACE}/regex_extractor:${IMAGE_TAG}; \
		docker tag ${DOCKER_NAMESPACE}/tensorboard_extract:${IMAGE_TAG} ${DOCKER_REPO}/${DOCKER_NAMESPACE}/tensorboard_extract:${IMAGE_TAG} && docker rmi ${DOCKER_NAMESPACE}/tensorboard_extract:${IMAGE_TAG}; \
		for i in $$(docker images --format '{{.Repository}}:{{.Tag}}' | grep ${DOCKER_REPO}/${DOCKER_NAMESPACE} | grep :${IMAGE_TAG} | grep -v '<none>'); do \
			echo "docker push $$i"; \
			docker push $$i; \
		done; \
		if [ "$(VM_TYPE)" = "none" ]; then \
			echo "The following images are now available:" ; \
			curl https://${DOCKER_REPO_USER}:${DOCKER_REPO_PASS}\@${DOCKER_REPO}/v2/_catalog ; \
		fi; \
	fi;

# TODO: setup-registry

create-registry:
	@kubectl create secret docker-registry regcred --docker-server=${DOCKER_REPO} --docker-username=${DOCKER_REPO_USER} --docker-password=${DOCKER_REPO_PASS} --docker-email=unknown@docker.io ; \
		cd ${DOCKER_REPO_DIR} ; \
		docker-compose up -d

# TODO Need to iterate over PVCs - currently only deletes static-volume-1
create-volumes:
	@kubectl delete --ignore-not-found=true pv/nfs pvc/static-volume-1 cm/static-volumes
	@cd bin; \
		./create_static_pv.sh; \
		./create_static_volumes.sh; \
		./create_static_volumes_config.sh;

deploy:           ## Deploy the services to Kubernetes
	@# deploy the stack via helm
	@echo Deploying services to Kubernetes. This may take a while.
	@if ! helm list > /dev/null 2>&1; then \
		echo 'Installing helm/tiller'; \
		helm init > /dev/null 2>&1; \
		sleep 3; \
	fi;
	@echo collecting existing pods
	@while kubectl get pods --all-namespaces | \
		grep -v RESTARTS | \
		grep -v Running | \
		grep 'alertmanager\|etcd0\|lcm\|restapi\|trainer\|trainingdata\|ui\|mongo\|prometheus\|pushgateway\|storage' > /dev/null; \
	do \
		sleep 1; \
	done
	@echo finding out which pods are running
	@while ! (kubectl get pods --all-namespaces | grep tiller-deploy | grep '1/1' > /dev/null); \
	do \
		sleep 1; \
	done
	@echo calling big command
	@set -o verbose; \
		existing=$$(helm list | grep ffdl | awk '{print $$1}' | head -n 1); \
		if [ "$$CI" = "true" ]; then \
			export helm_params='--set prometheus.deploy=false'; \
		else \
			export helm_params='--set lcm.shared_volume_storage_class=${SHARED_VOLUME_STORAGE_CLASS},has_static_volumes=${HAS_STATIC_VOLUMES},learner.docker_namespace=${DOCKER_NAMESPACE},docker.namespace=${DOCKER_NAMESPACE},learner.tag=${IMAGE_TAG},docker.pullPolicy=${DOCKER_PULL_POLICY},docker.registry=${DOCKER_REPO},trainer.version=${IMAGE_TAG},restapi.version=${IMAGE_TAG},lcm.version=${IMAGE_TAG},trainingdata.version=${IMAGE_TAG},databroker.tag=${IMAGE_TAG},databroker.version=${IMAGE_TAG},webui.version=${IMAGE_TAG}'; \
		fi; \
		(if [ -z "$$existing" ]; then \
			echo "Deploying the stack via Helm. This will take a while."; \
			helm --debug install $$helm_params . > /dev/null; \
			sleep 10; \
		else \
			echo "Upgrading existing Helm deployment ($$existing). This will take a while."; \
			helm upgrade $$helm_params $$existing . > /dev/null; \
		fi) & pid=$$!; \
		sleep 5; \
		while kubectl get pods --all-namespaces | \
			grep -v RESTARTS | \
			grep -v Running | \
			grep 'alertmanager\|etcd0\|lcm\|restapi\|trainer\|trainingdata\|ui\|mongo\|prometheus\|pushgateway\|storage'; \
		do \
			sleep 5; \
		done; \
		existing=$$(helm list | grep ffdl | awk '{print $$1}' | head -n 1); \
		for i in $$(seq 1 10); do \
			status=`helm status $$existing | grep STATUS:`; \
			echo $$status; \
			if echo "$$status" | grep "DEPLOYED" > /dev/null; then \
				kill $$pid > /dev/null 2>&1; \
				exit 0; \
			fi; \
			sleep 3; \
		done; \
		exit 0
	@echo done with big command
	@echo Initializing...
	@# wait for pods to be ready
	@while kubectl get pods --all-namespaces | \
		grep -v RESTARTS | \
		grep -v Running | \
		grep 'alertmanager\|etcd0\|lcm\|restapi\|trainer\|trainingdata\|ui\|mongo\|prometheus\|pushgateway\|storage' > /dev/null; \
	do \
		sleep 5; \
	done
	@echo initialize monitoring dashboards
	@if [ "$$CI" != "true" ]; then bin/grafana.init.sh; fi
	@echo
	@echo System status:
	@make status

undeploy:         ## Undeploy the services from Kubernetes
	@# undeploy the stack
	@existing=$$(helm list | grep ffdl | awk '{print $$1}' | head -n 1); \
		(if [ ! -z "$$existing" ]; then echo "Undeploying the stack via helm. This may take a while."; helm delete "$$existing"; echo "The stack has been undeployed."; fi) > /dev/null;

$(addprefix undeploy-, $(SERVICES)): undeploy-%: %
	@SERVICE_NAME=$< make .undeploy-service

.undeploy-service:
	@echo deleting $(SERVICE_NAME)
	(kubectl delete deploy,svc,statefulset --selector=service="ffdl-$(SERVICE_NAME)")

rebuild-and-deploy-lcm: build-lcm docker-build-lcm undeploy-lcm deploy

rebuild-and-deploy-trainer: build-trainer docker-build-trainer undeploy-trainer deploy

status:           ## Print the current system status and service endpoints
	@tiller=s; \
		status_kube=$$(kubectl config current-context) && status_kube="Running (context '$$status_kube')" || status_kube="n/a"; \
		echo "Kubernetes:\t$$status_kube"; \
		node_ip=$$(make --no-print-directory kubernetes-ip); \
		status_tiller=$$(helm list 2> /dev/null) && status_tiller="Running" || status_tiller="n/a"; \
		echo "Helm/Tiller:\t$$status_tiller"; \
		status_ffdl=$$(helm list 2> /dev/null | grep ffdl | awk '{print $$1}' | head -n 1) && status_ffdl="Running ($$(helm status "$$status_ffdl" 2> /dev/null | grep STATUS:))" || status_ffdl="n/a"; \
		echo "FfDL Services:\t$$status_ffdl"; \
		port_api=$$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}' 2> /dev/null) && status_api="Running (http://$$node_ip:$$port_api)" || status_api="n/a"; \
		echo "REST API:\t$$status_api"; \
		port_ui=$$(kubectl get service ffdl-ui -o jsonpath='{.spec.ports[0].nodePort}' 2> /dev/null) && status_ui="Running (http://$$node_ip:$$port_ui/#/login?endpoint=$$node_ip:$$port_api&username=test-user)" || status_ui="n/a"; \
		echo "Web UI:\t\t$$status_ui"; \
		status_grafana=$$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}' 2> /dev/null) && status_grafana="Running (http://$$node_ip:$$status_grafana) (login: admin/admin)" || status_grafana="n/a"; \
		echo "Grafana:\t$$status_grafana"

vagrant:          ## Configure local Kubernetes cluster in Vagrant
	vagrant up

minikube:         ## Configure Minikube (local Kubernetes)
  ifeq ($(filter $(UNAME),Linux Darwin),)
	@echo "Sorry, currently only supported for Mac OS and Linux"; exit 1
  endif
	@which minikube > /dev/null || (echo Please install Minikube; exit 1)
	@minikube ip > /dev/null 2>&1 || ( \
		echo "Starting up Minikube"; \
		minikube start --insecure-registry 9.0.0.0/8 --insecure-registry 10.0.0.0/8 \
				--cpus $(MINIKUBE_CPUS) \
				--memory $(MINIKUBE_RAM) \
				--vm-driver=$(MINIKUBE_DRIVER) > /dev/null; \
		sleep 5; \
	)
	@set -o verbose; \
	minikube ip > /dev/null 2>&1 && ( \
		if [ "$(SET_LOCAL_ROUTES)" == "1" ]; then \
			echo "Update local network routes, using '$(MINIKUBE_BRIDGE)' as network bridge (you may be prompted for your sudo password)"; \
			if [ "$(UNAME)" == "Linux" ]; then \
				sudo route -n add -net 10.0.0.0/24 gw $(minikube ip); \
			elif [ "$(UNAME)" == "Darwin" ]; then \
				sudo route -n delete 10.0.0.0/24 > /dev/null 2>&1; \
				sudo route -n add 10.0.0.0/24 $$(minikube ip) > /dev/null 2>&1; \
				member_interface=$$(ifconfig $(MINIKUBE_BRIDGE) | grep member | awk '{print $$2}' | head -1); \
				if [ "$$member_interface" != "" ]; then sudo ifconfig $(MINIKUBE_BRIDGE) -hostfilter $$member_interface; fi; \
			fi; \
			kubectl get configmap/kube-dns --namespace kube-system -o yaml > /tmp/kube-dns.cfg; \
			grep upstreamNameservers /tmp/kube-dns.cfg > /dev/null || (echo 'data:' >> /tmp/kube-dns.cfg; \
				echo '  upstreamNameservers: '"'"'["8.8.8.8"]'"'" >> /tmp/kube-dns.cfg); \
			kubectl replace -f /tmp/kube-dns.cfg > /dev/null; \
		fi; \
	)

minikube-undo:    ## Revert changes applied by the "minikube" target (e.g., local network routes)
	@minikube ip > /dev/null 2>&1 && ( \
		sudo route -n delete 10.0.0.0/24 > /dev/null 2>&1; \
	) || true

# Build targets

gen-certs:
	cd commons/certs ; \
	./generate.sh ; \
	./install.sh

del-certs:
	rm -rf ./metrics/docker/certs; \
	rm -rf ./metrics/log_collectors/training_data_service_client/certs; \
	rm -rf ./restapi/certs; \
	rm -rf ./lcm/certs; \
	rm -rf ./jobmonitor/certs; \
	rm -rf ./trainer/docker/certs; \
	rm -f ./commons/certs/ca.*; \
	rm -f ./commons/certs/client.*; \
	rm -f ./commons/certs/server.*

$(addprefix build-, $(SERVICES)): build-%: %
	@SERVICE_NAME=$< BINARY_NAME=main make .build-service

build-cli:
	cd ./cli/ && (CGO_ENABLED=0 go build -ldflags "-s -w" -a -installsuffix cgo -o bin/ffdl-osx; \
		CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o bin/ffdl-linux)

build-cli-opt:
	make build-cli
	cd ./cli/; upx --brute bin/ffdl-osx; \
		docker run -v `pwd`:/ffdl-cli ubuntu bash -c 'cd /ffdl-cli; apt-get update; apt-get install -y upx-ucl; upx --brute bin/ffdl-linux'

# Docker build targets

$(addprefix docker-build-, $(IMAGES)): docker-build-%: %
	@IMAGE_NAME=$< make .docker-build

docker-build-ui:
	mkdir -p build; test -e dashboard || (git clone $(UI_REPO) build/ffdl-ui; ln -s build/ffdl-ui dashboard)
	(cd dashboard && (if [ "$(VM_TYPE)" = "minikube" ]; then eval $$(minikube docker-env); fi; \
		docker build -q -t $(DOCKER_REPO)/$(DOCKER_NAMESPACE)/$(IMAGE_NAME_PREFIX)ui:$(IMAGE_TAG) .; \
		(test ! `which docker-squash` || docker-squash -t $(DOCKER_REPO)/$(DOCKER_NAMESPACE)/$(IMAGE_NAME_PREFIX)ui $(DOCKER_REPO)/$(DOCKER_NAMESPACE)/$(IMAGE_NAME_PREFIX)ui)))

docker-build-base:
	(cd etc/dlaas-service-base; make build)

docker-build-logcollectors:
	if [ "$(VM_TYPE)" = "minikube" ]; then \
		eval $$(minikube docker-env); \
	fi; \
	cd metrics/log_collectors && IMAGE_TAG=$(IMAGE_TAG) && make build-simple_log_collector build-emetrics_file build-regex_extractor build-tensorboard-all


test-push-data:      ## Test
	@# Pushes test data to S3 buckets.
	@echo Pushing test data.
	@s3_ip=$$(kubectl get po/storage-0 -o=jsonpath='{.status.hostIP}'); \
        	s3_port=$$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}'); \
        	s3_url=http://$$s3_ip:$$s3_port; \
		export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1; \
        	s3cmd="aws --endpoint-url=$$s3_url s3"; \
        	$$s3cmd mb s3://tf_training_data > /dev/null; \
        	$$s3cmd mb s3://tf_trained_model > /dev/null; \
        	$$s3cmd mb s3://mnist_lmdb_data > /dev/null; \
        	$$s3cmd mb s3://dlaas-trained-models > /dev/null; \
        	for file in t10k-images-idx3-ubyte.gz t10k-labels-idx1-ubyte.gz train-images-idx3-ubyte.gz train-labels-idx1-ubyte.gz; do \
               		test -e $(TMPDIR)/$$file || wget -q -O $(TMPDIR)/$$file http://yann.lecun.com/exdb/mnist/$$file; \
               		$$s3cmd cp $(TMPDIR)/$$file s3://tf_training_data/$$file > /dev/null; \
        	done; \
        	for phase in train test; do \
               		for file in data.mdb lock.mdb; do \
                      		tmpfile=$(TMPDIR)/$$phase.$$file; \
                      		test -e $$tmpfile || wget -q -O $$tmpfile https://github.com/albarji/caffe-demos/raw/master/mnist/mnist_"$$phase"_lmdb/$$file; \
                      		$$s3cmd cp $$tmpfile s3://mnist_lmdb_data/$$phase/$$file > /dev/null; \
               		done; \
        	done;

test-submit:      ## Submit test training job
	@# make sure the buckets with training data exist
	@echo Downloading Docker images and test training data. This may take a while.
	@if [ "$(VM_TYPE)" = "minikube" ]; then \
			eval $(minikube docker-env); docker images | grep tensorflow | grep latest > /dev/null || docker pull tensorflow/tensorflow > /dev/null; \
		fi
	@node_ip=$$(make --no-print-directory kubernetes-ip); \
                s3_ip=$$(kubectl get po/storage-0 -o=jsonpath='{.status.hostIP}'); \
		s3_port=$$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}'); \
		s3_url=http://$$s3_ip:$$s3_port; \
		echo "Submitting example training job ($(TEST_SAMPLE))"; \
		restapi_port=$$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}'); \
		restapi_url=http://$$node_ip:$$restapi_port; \
		echo S3 URL: $$s3_url  REST URL: $$restapi_url; \
		export DLAAS_URL=http://$$node_ip:$$restapi_port; export DLAAS_USERNAME=$(TEST_USER); export DLAAS_PASSWORD=test; \
		echo Executing in etc/examples/$(TEST_SAMPLE): DLAAS_URL=$$DLAAS_URL DLAAS_USERNAME=$$DLAAS_USERNAME DLAAS_PASSWORD=test $(CLI_CMD) train manifest.yml . ; \
		cp etc/examples/tf-model/manifest.yml etc/examples/tf-model/manifest_testrun.yml ; \
		sed -i '' -e "s/user_name: test/user_name: ${AWS_ACCESS_KEY_ID}/g" etc/examples/tf-model/manifest_testrun.yml ; \
		sed -i '' -e "s/password: test/password: ${AWS_SECRET_ACCESS_KEY}/g" etc/examples/tf-model/manifest_testrun.yml ; \
		cat etc/examples/tf-model/manifest_testrun.yml ; \
		(cd etc/examples/$(TEST_SAMPLE); pwd; $(CLI_CMD) train manifest_testrun.yml .); \
		rm -f etc/examples/tf-model/manifest_testrun.yml ; \
		echo Test job submitted. Track the status via '"'DLAAS_URL=$$DLAAS_URL DLAAS_USERNAME=$(TEST_USER) DLAAS_PASSWORD=test $(CLI_CMD) list'"'. ;

test-localmount-submit:      ## Submit test training job
	@# make sure the buckets with training data exist
	@echo Downloading Docker images and test training data. This may take a while.
	@if [ "$(VM_TYPE)" = "minikube" ]; \
		then \
			echo "Checking if we have a tensorflow image, and maybe pulling..."; \
			eval $(minikube docker-env); \
			docker images | grep tensorflow | grep latest > /dev/null || docker pull tensorflow/tensorflow > /dev/null; \
			echo "Done checking if we have a tensorflow image"; \
		fi
	@node_ip=$$(make --no-print-directory kubernetes-ip); \
		echo "Submitting example training job ($(TEST_SAMPLE))"; \
		restapi_port=$$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}'); \
		export DLAAS_URL=http://$$node_ip:$$restapi_port; export DLAAS_USERNAME=$(TEST_USER); export DLAAS_PASSWORD=test; \
		(cd etc/examples/$(TEST_SAMPLE); $(CLI_CMD) train manifest-hostmount.yml .); \
		echo Test job submitted. Track the status via '"'DLAAS_URL=$$DLAAS_URL DLAAS_USERNAME=$(TEST_USER) DLAAS_PASSWORD=test $(CLI_CMD) list'"'. ; \
		sleep 10; \
		for i in $$(seq 1 50); do output=$$($(CLI_CMD) list 2>&1 | grep training-); \
			if echo $$output | grep 'FAILED'; then echo 'Job failed'; exit 1; fi; \
			if echo $$output | grep 'COMPLETED'; then echo 'Job completed'; exit 0; fi; \
			echo $$output; \
			sleep 20; \
		done;

#			 || \
#			($(CLI_CMD) list; \
#				kubectl get pods | grep learner- | awk '{print $$1}' | xargs -I '{}' kubectl describe pod '{}'; \
#				kubectl get pods | grep learner- | awk '{print $$1}' | xargs -I '{}' kubectl logs '{}' -c learner; \
#				exit 0);

# Helper targets

test-s3:
	node_ip=$$(make --no-print-directory kubernetes-ip); \
	s3_port=$$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}'); \
	s3_url=http://$$node_ip:$$s3_port; \
	s3_url="https://s3.us-west.objectstorage.uat.softlayer.net"; \
	export AWS_DEFAULT_REGION=us-west-1; \
	s3cmd="aws --endpoint-url=$$s3_url s3"; \
	echo "s3cmd=$$s3cmd"; \
	$$s3cmd ls


.build-service:
	(cd ./$(SERVICE_NAME)/ && (test ! -e main.go || CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -installsuffix cgo -o bin/$(BINARY_NAME)))

.docker-build:
	(full_img_name=$(IMAGE_NAME_PREFIX)$(IMAGE_NAME); \
		cd ./$(IMAGE_DIR)/ && ( \
		if [ "$(VM_TYPE)" = "minikube" ]; then \
			eval $$(minikube docker-env); \
		fi; \
		docker build -q -t $(DOCKER_REPO)/$(DOCKER_NAMESPACE)/$$full_img_name:$(IMAGE_TAG) .))

kubernetes-ip:
	@if [ "$$CI" = "true" ]; then kubectl get nodes -o jsonpath='{ .items[0].status.addresses[?(@.type=="InternalIP")].address }'; \
		elif [ "$(VM_TYPE)" = "vagrant" ]; then \
			node_ip_line=$$(vagrant ssh master -c 'ifconfig eth1 | grep "inet "' 2> /dev/null); \
			node_ip=$$(echo $$node_ip_line | sed "s/.*inet \([^ ]*\) .*/\1/"); \
			echo $$node_ip; \
		elif [ "$(VM_TYPE)" = "minikube" ]; then \
			echo $$(minikube ip); \
		elif [ "$(VM_TYPE)" = "ibmcloud" ]; then \
			echo $$(bx cs workers $(CLUSTER_NAME) | grep Ready | awk '{ print $$2;exit }'); \
		elif [ "$(VM_TYPE)" = "none" ]; then \
			echo "$(PUBLIC_IP)"; \
		fi

install-minikube-in-ci:
	@echo "Starting local kubernetes cluster (minikube)"
	@curl -s -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/$(CI_KUBECTL_VERSION)/bin/linux/amd64/kubectl && \
		chmod +x kubectl && sudo mv kubectl /usr/local/bin/
	@curl -s -Lo minikube https://storage.googleapis.com/minikube/releases/$(CI_MINIKUBE_VERSION)/minikube-linux-amd64 && \
		chmod +x minikube && sudo mv minikube /usr/local/bin/
	@sudo minikube start --vm-driver=none
	@minikube update-context
	@JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; \
		until kubectl get nodes -o jsonpath="$$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
	@while ! kubectl get svc/kubernetes > /dev/null; do sleep 1; done
	@while ! kubectl get rc/kubernetes-dashboard --namespace=kube-system > /dev/null 2>&1; do sleep 1; done
	@kubectl delete  --namespace=kube-system rc/kubernetes-dashboard svc/kubernetes-dashboard || true

.PHONY: build deploy controller databroker_objectstorage databroker_s3
