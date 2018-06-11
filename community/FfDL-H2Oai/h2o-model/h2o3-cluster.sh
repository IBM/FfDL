pip install -q awscli
export AWS_ACCESS_KEY_ID=test; export AWS_SECRET_ACCESS_KEY=test; export AWS_DEFAULT_REGION=us-east-1;
export s3cmd="aws --endpoint-url=http://$S3_SERVICE_HOST:$S3_SERVICE_PORT s3"
$s3cmd mb s3://flatfiles
echo $(hostname --ip-address):54321 >> $LEARNER_ID.txt
$s3cmd cp $LEARNER_ID.txt s3://flatfiles/$SERVICE_NAME_1/$LEARNER_ID.txt
lines=$($s3cmd ls s3://flatfiles/$SERVICE_NAME_1/ | wc -l);
while [ "$lines" != "$NUM_LEARNERS" ]; do sleep 5; lines=$($s3cmd ls s3://flatfiles/$SERVICE_NAME_1/ | wc -l); done;
mkdir flatfiles;
$s3cmd sync s3://flatfiles/$SERVICE_NAME_1 flatfiles/;
cat flatfiles/* > flatfile.txt;
java -jar /opt/h2o.jar -flatfile flatfile.txt -name h2oCluster &
while [ "$nodes" != "$NUM_LEARNERS" ]; do sleep 30; nodes=$(curl -s http://$(hostname --ip-address):54321/3/Cloud.json | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["'cloud_size'"]';); done;
