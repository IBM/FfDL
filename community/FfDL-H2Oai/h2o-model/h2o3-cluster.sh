echo $(hostname --ip-address):54321 >> $LEARNER_ID.txt
mkdir ${RESULT_DIR}/flatfiles
cp $LEARNER_ID.txt ${RESULT_DIR}/flatfiles/$LEARNER_ID.txt
lines=$(ls ${RESULT_DIR}/flatfiles/ | wc -l);
while [ "$lines" != "$NUM_LEARNERS" ]; do sleep 5; lines=$(ls ${RESULT_DIR}/flatfiles/ | wc -l); done;
cat ${RESULT_DIR}/flatfiles/* > flatfile.txt;
java -jar /opt/h2o.jar -flatfile flatfile.txt -name h2oCluster &
while [ "$nodes" != "$NUM_LEARNERS" ]; do sleep 30; nodes=$(curl -s http://$(hostname --ip-address):54321/3/Cloud.json | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["'cloud_size'"]';); done;
