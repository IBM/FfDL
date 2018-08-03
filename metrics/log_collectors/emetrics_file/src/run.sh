#!/usr/bin/env bash
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


export LEARNER_ID=$((${DOWNWARD_API_POD_NAME##*-} + 1)) ;
echo "* * * * * AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY \
timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL s3 sync \
$LOG_DIR s3://$RESULT_STORE_OBJECTID"> crontab.txt && \
echo "* * * * * (sleep 30 && AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY \
 timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL s3 sync \
 $LOG_DIR s3://$RESULT_STORE_OBJECTID)">> crontab.txt && crontab crontab.txt && rm -f crontab.txt
service cron start
python3 tail_em_from_emfile.py
echo "Saving final logs for $TRAINING_ID : " && time AWS_ACCESS_KEY_ID=$RESULT_STORE_USERNAME \
AWS_SECRET_ACCESS_KEY=$RESULT_STORE_APIKEY timeout -s 3 20 /usr/local/bin/aws --endpoint-url=$RESULT_STORE_AUTHURL \
s3 sync $LOG_DIR s3://$RESULT_STORE_OBJECTID >&2
ERROR_CODE=$?
echo "echo aws s3 exit code $ERROR_CODE"
echo "Writing $ERROR_CODE to $JOB_STATE_DIR" >&2
echo $ERROR_CODE > $JOB_STATE_DIR/lc.exit

echo "done writing state file to nfs"

# exit $ERROR_CODE
sleep 180

echo "exiting log-collector instead of being killed"

# this probably won't execute?
exit $ERROR_CODE