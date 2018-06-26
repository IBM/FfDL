grafana_port=$(kubectl get service grafana -o jsonpath='{.spec.ports[0].nodePort}')
ui_port=$(kubectl get service ffdl-ui -o jsonpath='{.spec.ports[0].nodePort}')
restapi_port=$(kubectl get service ffdl-restapi -o jsonpath='{.spec.ports[0].nodePort}')
s3_port=$(kubectl get service s3 -o jsonpath='{.spec.ports[0].nodePort}')
ui_pod=$(kubectl get pods | grep ffdl-ui | awk '{print $1}')
restapi_pod=$(kubectl get pods | grep ffdl-restapi | awk '{print $1}')
grafana_pod=$(kubectl get pods | grep prometheus | awk '{print $1}')
kubectl port-forward pod/$ui_pod $ui_port:8080 &
kubectl port-forward pod/$restapi_pod $restapi_port:8080 &
kubectl port-forward pod/$grafana_pod $grafana_port:3000 &
kubectl port-forward pod/storage-0 $s3_port:4572 &
