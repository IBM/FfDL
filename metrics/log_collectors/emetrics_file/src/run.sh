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


: ${TRAININGLOGS:="$JOB_STATE_DIR/latest-log"}
#export TRAININGLOGS=$JOB_STATE_DIR/logs/training-log.txt
: ${SHOULD_CONNECT:="--send"}

echo "Training logs: $TRAININGLOGS"
echo "SHOULD_CONNECT: $SHOULD_CONNECT"

while [ ! -f $TRAININGLOGS ]
do
  sleep 1
done

python3 tail_to_tds.py &

python3 tail_em_from_emfile.py ${SHOULD_CONNECT}
