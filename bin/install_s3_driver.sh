#!/usr/bin/env bash
# NOTE: This is an experimental(!) script to fully automatically setup the s3 driver. Proceed with caution.

# Make sure user has sudo privileges...
if [[ ! $(sudo echo 0) ]]; then exit; fi
# ...and is logged in
bx login -a https://api.ng.bluemix.net -sso
bx cr login

if [[ -z "${GOPATH}" ]]; then
echo "Critical environment variable not defined. Exiting."
exit 1
fi

# Install dependencies
sudo apt install -y build-essential libfuse2 libxml2
go get github.com/gin-gonic/gin

# Clone and build repository
mkdir $GOPATH/bin
mkdir -p $GOPATH/src/github.ibm.com/alchemy-containers && cd $_
git clone git@github.ibm.com:alchemy-containers/armada-storage-s3fs-plugin.git
cd ./armada-storage-s3fs-plugin
make
make provisioner
make driver

# Deploy with Helm
cd ./deploy/helm/
sed -i 's/<REGISTRY_URL>/registry.ng.bluemix.net/g' ibmcloud-object-storage-plugin/values.yaml
helm init --wait
helm install ./ibmcloud-object-storage-plugin
kubectl get storageclass | grep s3
kubectl get pods -n kube-system | grep object-storage

# Prepare files
mkdir -p $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-addendum/ && cd $_
docker pull registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-plugin:128
docker pull registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-driver:220
docker save registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-plugin:128 > ibmcloud-object-storage-plugin.tar
docker save registry.ng.bluemix.net/armada-master/ibmcloud-object-storage-driver:220 > ibmcloud-object-storage-driver.tar

cat <<_EOF > dummy.cm
{
  "account_id": "kube-user",
  "customer_id": "kube-user",
  "cluster_id": "private12345",
  "cluster_name": "private-cluster",
  "cluster_type": "private",
  "datacenter": "local"
}
_EOF
cat <<_EOF > secret.yaml
apiVersion: v1
kind: Secret
type: ibm/ibmc-s3fs
metadata:
  name: test-secret
  namespace: default
data:
  access-key: <MY_ACCESS_KEY>
  secret-key: <MY_SECRET_KEY>
_EOF
cat <<_EOF > pvc.yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: s3fs-test-pvc
  namespace: default
  annotations:
    volume.beta.kubernetes.io/storage-class: "ibmc-s3fs-standard"
    ibm.io/auto-create-bucket: "false"
    ibm.io/auto-delete-bucket: "false"
    ibm.io/bucket: "<YOUR_BUCKET_NAME>"
    ibm.io/endpoint: "https://s3-api.us-geo.objectstorage.softlayer.net"
    ibm.io/region: "us-standard"
    ibm.io/secret-name: "test-secret"
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 8Gi
_EOF
cat <<_EOF > pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: s3fs-test-pod
  namespace: default
spec:
  containers:
  - name: s3fs-test-container
    image: anaudiyal/infinite-loop
    volumeMounts:
    - mountPath: "/mnt/s3fs"
      name: s3fs-test-volume
  volumes:
  - name: s3fs-test-volume
    persistentVolumeClaim:
      claimName: s3fs-test-pvc
_EOF

# Import plugin and driver into every Kubernetes node
declare -a arrNodes=($(docker ps --format '{{.Names}}' | grep "kube-node-\|kube-master"))
for node in "${arrNodes[@]}"
do
docker cp ibmcloud-object-storage-driver.tar $node:/
docker cp ibmcloud-object-storage-plugin.tar $node:/
docker exec -i $node /bin/bash <<'_EOF'
docker load --input ibmcloud-object-storage-driver.tar
docker load --input ibmcloud-object-storage-plugin.tar
apt install -y libfuse2 libxml2
_EOF
done

# Create configmap and install S3 plugins
kubectl create configmap -n  kube-system cluster-info --from-file=cluster-config.json=./dummy.cm
cd $GOPATH/src/github.ibm.com/alchemy-containers/armada-storage-s3fs-plugin/deploy/helm
helm install ./ibmcloud-object-storage-plugin-security
helm install ./ibmcloud-object-storage-plugin
# Delete storage plugin to restart it
kubectl delete $(kubectl get pods -n kube-system -o name | grep ibmcloud-object-storage-plugin-) -n kube-system
echo "Basic S3 driver deployment complete. Storage Information:"
kubectl get storageclass | grep s3
echo "-------"
kubectl get pods -n kube-system | grep object-storage
echo "You can now insert your credentials into secret.yaml and your bucket name into pvc.yaml. Please encode your keys with with echo -n <key> | base64 before doing so."
echo "You can then run an example pod that mounts the specified S3 bucket as follows:"
echo "kubectl apply -f secret.yaml"
echo "kubectl apply -f pvc.yaml"
echo "kubectl apply -f pod.yaml"
echo "kubectl exec -it s3fs-test-pod bash"
echo "    df -Th | grep s3"
echo "    ls /mnt/s3fs/"
