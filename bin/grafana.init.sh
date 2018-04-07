#!/bin/bash
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
# Note: This script should be called from the top-level working directory, i.e., bin/grafana.init.sh

# get grafana service IP
if [ "$VM_TYPE" == "ibmcloud" ]; then
	node_ip=$(bx cs workers $CLUSTER_NAME | grep Ready | awk '{ print $2;exit }')
elif [ "$VM_TYPE" == "minikube" ]; then
	node_ip=$(minikube ip)
elif [ "$VM_TYPE" == "vagrant" ]; then
	node_ip_line=$(vagrant ssh master -c 'ifconfig eth1 | grep "inet "' 2> /dev/null)
	node_ip=$(echo $node_ip_line | sed "s/.*inet \([^ ]*\) .*/\1/")
elif [ "$VM_TYPE" == "none" ]; then
	node_ip=$PUBLIC_IP
fi
grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
grafana_url="http://$node_ip:$grafana_port"

# wait until the service is up
while ! curl -q $grafana_url > /dev/null 2>&1 ; do sleep 2; done;

# create data source
curl -q -u admin:admin -H 'Content-Type: application/json' -XPOST $grafana_url/api/datasources \
  -d '{"name":"prom","type":"prometheus","url":"http://localhost:9090","access":"proxy","basicAuth":false}' > /dev/null 2>&1

# dashboards
curl -q -u admin:admin -H 'Content-Type: application/json' -XPOST $grafana_url/api/dashboards/db \
  -d @etc/dashboards.json > /dev/null 2>&1
