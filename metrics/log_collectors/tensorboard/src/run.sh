#!/usr/bin/env bash

export PYTHONPATH=${PYTHONPATH}:$PWD

export LEARNER_ID=$((${DOWNWARD_API_POD_NAME##*-} + 1)) ;
echo "* * * * * AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY \
timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL s3 sync \
$LOG_DIR s3://$RESULT_STORE_OBJECTID"> crontab.txt && \
echo "* * * * * (sleep 30 && AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY \
 timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL s3 sync \
 $LOG_DIR s3://$RESULT_STORE_OBJECTID)">> crontab.txt && crontab crontab.txt && rm -f crontab.txt
service cron start
python extract_tb.py
echo "Saving final logs for $TRAINING_ID : " && time AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME \
AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL \
s3 sync $LOG_DIR s3://$RESULT_STORE_OBJECTID
ERROR_CODE=$?
echo "echo aws s3 exit code $ERROR_CODE"
echo "deleting contents of logdir"
rm -rf $LOG_DIR/tb/*
echo "Writing $ERROR_CODE to $JOB_STATE_DIR" >&2
echo $ERROR_CODE > $JOB_STATE_DIR/lc.exit

echo "done writing state file to nfs"

# exit $ERROR_CODE
sleep 180

echo "exiting log-collector instead of being killed"

# this probably won't execute?
exit $ERROR_CODE