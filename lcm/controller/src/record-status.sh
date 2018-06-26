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


# Set or update a status node.

# Expected inputs:
#   $1 arg: the ZNode path.
#   $2 arg: the value to record in the ZNode.
#   $3 arg (optional): if value is "create" then create the ZNode.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

function recordStatusInEtcd {

	ZNODE_PATH=$1
	STATUS_STRING=${2:-""}

	# Append timestamp to node path, as we are storing the history of nodes
	# NOTE: The command below only provides the desired nanosecond precision on some systems
	#       (including Ubuntu), but on other systems (e.g., Alpine) it only has seconds precision.
	nano_time=$(date "+%s%N")
	ZNODE_PATH=$ZNODE_PATH/$nano_time

	STATUS_VALUE=$STATUS_STRING
	# STATUS_VALUE could be a JSON string like {"status":"FAILED",...}
	if [[ "$STATUS_VALUE" == "{"* ]]; then
		STATUS_VALUE=$( echo $STATUS_VALUE | sed -e 's/{.*"status"\s*:\s*"\([^"]*\)".*}/\1/g' )
	fi

	# Update the ZNode value using with_backoff logic and finite tries for intermediate steps but with infinite retries for final steps
	case "${STATUS_VALUE}" in
	    (COMPLETED)
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (FAILED)
	        updateMetricsOnTrainingFailure "FAILED" &
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (HALTED)
	        updateMetricsOnTrainingFailure "HALTED" &
	        infinite_exp_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"
	        ;;
	    (*)
	        with_backoff runEtcdCommand put ${ZNODE_PATH} "${STATUS_STRING}"

	esac

}
