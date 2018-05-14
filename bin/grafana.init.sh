#!/bin/bash

# Note: This script should be called from the top-level working directory, i.e., bin/grafana.init.sh

# get grafana service IP
echo "\$VM_TYPE == \"$VM_TYPE\""
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

[ -z ${node_ip} ] && echo "Can't get node_ip for grafana, \$VM_TYPE == \"$VM_TYPE\""

grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
grafana_url="http://$node_ip:$grafana_port"

echo "wait until the grafana service is up (grafana_url=${grafana_url})"
while ! curl -q $grafana_url > /dev/null 2>&1 ; do sleep 2; done;

echo create data source
curl -q -u admin:admin -H 'Content-Type: application/json' -XPOST $grafana_url/api/datasources \
  -d '{"name":"prom","type":"prometheus","url":"http://localhost:9090","access":"proxy","basicAuth":false}' > /dev/null 2>&1

echo dashboards
curl -q -u admin:admin -H 'Content-Type: application/json' -XPOST $grafana_url/api/dashboards/db \
  -d @etc/dashboards.json > /dev/null 2>&1
