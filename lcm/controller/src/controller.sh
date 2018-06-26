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

#   $1 arg: the ZNode path.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"
source "$SCRIPTDIR/record-status.sh"

: ${JOB_STATE_DIR:="/job"}

env | sort

# Hold the current state in this file.
# Valid states:
# - INIT: initial state
# - DOWNLOADING: downloading model and training data
# - PROCESSING: do training
# - LC_WAIT_ON_SUCCESS: job done, wait for log-collector to finish; expect to transition to STORING_ON_SUCCESS state
# - LC_WAIT_ON_FAILURE: job failed, wait for log-collector to finish; expect to transition to STORING_ON_FAILURE
# - LC_WAIT_ON_HALTED: job halted, wait for log-collector to finish; expect to transition to STORING_ON_HALTED
# - STORING_ON_SUCCESS: uploading results; no errors so far; expect to transition to COMPLETED state after
# - STORING_ON_FAILURE: uploading results; had errors already; expect to transition to FAILED state after
# - STORING_ON_HALTED: uploading results; triggerd by halt command; expect to transition to HALTED state after
# - COMPLETED: final successful state
# - FAILED: final error state
state_file="$JOB_STATE_DIR/current_state"

# The containers started in each state.
declare -A stateContainers
stateContainers[INIT]=""
stateContainers[DOWNLOADING]="load-model load-data"
stateContainers[PROCESSING]="learner"
stateContainers[LC_WAIT_ON_SUCCESS]=""
stateContainers[LC_WAIT_ON_FAILURE]=""
stateContainers[LC_WAIT_ON_HALTED]=""
stateContainers[STORING_ON_SUCCESS]="store-results store-logs"
stateContainers[STORING_ON_FAILURE]="store-results store-logs"
stateContainers[STORING_ON_HALTED]="store-results store-logs"
stateContainers[COMPLETED]=""
stateContainers[FAILED]=""
stateContainers[FINAL]=""

# The presence of this file indicates a halt has been requested.
halt_file="$JOB_STATE_DIR/halt"

lc_exit_file="$JOB_STATE_DIR/lc.exit"

# Note that associative arrays are for bash 4 only
declare -A lc_transitions
lc_transitions[LC_WAIT_ON_SUCCESS]=STORING_ON_SUCCESS
lc_transitions[LC_WAIT_ON_FAILURE]=STORING_ON_FAILURE
lc_transitions[LC_WAIT_ON_HALT]=STORING_ON_HALT

user_log_file="$JOB_STATE_DIR/logs/training-log.txt"

TIME_TO_SLEEP_AFTER_USER_LOG=2
TIME_TO_SLEEP_FOR_LOG_COLLECTOR=240

# Initialize the current state if it's not already set.
function init() {
    if [ ! -f "$state_file" ]; then
        echo Initializing
        echo INIT > "$state_file"
    fi

    # Create file for training logs.
    mkdir -p "$JOB_STATE_DIR/logs"
    touch "$JOB_STATE_DIR/logs/training-log.txt"
    ln -sf "$JOB_STATE_DIR/logs/training-log.txt" "$JOB_STATE_DIR/latest-log"

    # Allow anyone to write to job state dir.
    # This allows learner processes (which don't run as root) to write to the directory.
    chmod -R 777 "$JOB_STATE_DIR"
}

# Cleanup after job is done.
function cleanup() {
    # Delete files in the job directory.
    # Keep the following control files:
    #   current_state: needed so we don't repeat this state machine.
    find "$JOB_STATE_DIR" -maxdepth 1 -mindepth 1 ! -name current_state -exec rm -vrf \{\} \;
}

# Check for creation of the /ZK_DIR/TRAINING_ID/halt znode
function checkForHaltZNode() {
    ZNODE_PATH="$JOB_BASE_PATH/halt"
    if infinite_exp_backoff runEtcdCommand watch $ZNODE_PATH ; then
        touch "$halt_file"
    fi
}

# Set $current_state variable to the current state
function getState() {
    current_state=$(cat "$state_file")
}


# Set current state to the value in $1
function setState() {
    new_state=$1
    getState
    echo "Transition state $current_state -> $new_state"
    echo $1 > "$state_file"
    startContainersForCurrentState
}

# Record job status with value in etcd as $1
function sendStatusUpdate() {
    value=$1
    recordStatusInEtcd "$JOB_LEARNER_ZNODE_STATUS_PATH" "$value"
}

function recordStatus() {
	status=$1
	error_code=${2:-""}
	message=${3:-""}
	timestamp=${4:-""}
	if [[ "$timestamp" = "" ]]; then
		# get timestamp as milliseconds since Unix epoch
		timestamp=$(date +%s%N | cut -b1-13)
	fi
	sendStatusUpdate '{"timestamp":"'$timestamp'","status":"'$status'","error_code":"'$error_code'","status_message":"'$message'"}'
}

function updateStatusTimestamp() {
	status=$1
	timestamp_file=$2
	if [ -f "$timestamp_file" ]; then
		timestamp=$(cat "$timestamp_file")
		recordStatus $status "" "" "$timestamp"
	fi
}

# Signal container named $1 to start.
function startContainer() {
    container_name=$1
    start_code_file="$JOB_STATE_DIR/$container_name.start"
    touch "$start_code_file"
}

# Signal containers for the current state.
function startContainersForCurrentState() {
    getState
    containers=${stateContainers[$current_state]}
    echo "Starting containers for state $current_state: $containers"
    for c in $containers; do
        startContainer $c
    done
}

# Sets $exit_code variable to the exit code of container $1
# If the container hasn't completed yet, set $exit_code to an empty string.
function getExitCode() {
    container_name=$1
    exit_code_file="$JOB_STATE_DIR/$container_name.exit"
    if [ -f "$exit_code_file" ]; then
        exit_code=$(cat "$exit_code_file")
    else
        exit_code=""
    fi
}

# Sends off a monitoring event via statsd to track the duration of certain phases
function pushDurationMetric() {
    if [[ $previous_state = DOWNLOADING ]] || [[ $previous_state = PROCESSING ]]; then
        end_time=$(date "+%s")
        if [[ "$start_time" != "" ]]; then
            duration=$(expr $end_time - $start_time)
            state_lowercase=$(echo "$previous_state" | tr '[:upper:]' '[:lower:]')
            pushMetrics "controller.phase.duration:$duration|h|#phase:$state_lowercase" &
        fi
    fi
}

previous_state=""

echo "Initiating job" >> $user_log_file

# State machine loop
init
checkForHaltZNode &
while true; do
    getState
    if [[ $current_state != $previous_state ]]; then

        pushDurationMetric
        start_time=$(date "+%s")

        echo "-- Current state: $current_state"
        previous_state=$current_state
    fi
    case "$current_state" in
        (INIT)
            # INIT -> DOWNLOADING always
            recordStatus DOWNLOADING
            setState DOWNLOADING
            ;;
        (DOWNLOADING)
            # DOWNLOADING -> PROCESSING if loading succeeds
            # DOWNLOADING -> FAILED if loading fails
            # DOWNLOADING -> HALTED if halt triggered

            getExitCode load-model; load_model_exit_code=$exit_code
            getExitCode load-data;  load_data_exit_code=$exit_code
            [[ -z "$load_model_exit_code" ]] || echo "load-model exit: $load_model_exit_code"
            [[ -z "$load_data_exit_code"  ]] || echo "load-data exit: $load_data_exit_code"

            if [[ ! -z "$load_model_exit_code" && "$load_model_exit_code" != "0" ]]; then
                # Load model failed.
                echo "Failed: load_model_exit_code: $load_model_exit_code" >> $user_log_file
                sleep ${TIME_TO_SLEEP_AFTER_USER_LOG}
                # Record error details. Code S301 indicates "failed to load model" (see jobmonitor for a list of error codes)
                recordStatus FAILED S301 $load_model_exit_code
                setState FAILED
            elif [[ ! -z "$load_data_exit_code" && "$load_data_exit_code" != "0" ]]; then
                # Load data failed.
                echo "Failed: load_data_exit_code: $load_data_exit_code" >> $user_log_file
                sleep ${TIME_TO_SLEEP_AFTER_USER_LOG}
                recordStatus FAILED S302 $load_data_exit_code
                setState FAILED
            elif [[ -f "$halt_file" ]]; then
                # User wants to halt the job. Skip processing and storing.
                echo "Halted: user requests halt job" >> $user_log_file
                sleep ${TIME_TO_SLEEP_AFTER_USER_LOG}
                recordStatus HALTED
                setState HALTED
            elif [[ "$load_model_exit_code" == "0" && "$load_data_exit_code" == "0" ]]; then
                # Both completed successfully.
                recordStatus PROCESSING
                setState PROCESSING
            fi
            ;;
        (PROCESSING)
            # PROCESSING -> STORING_ON_SUCCESS if learner succeeds
            # PROCESSING -> STORING_ON_FAILURE if learner fails
            # PROCESSING -> HALTED if halt triggered

            getExitCode learner; learner_exit_code=$exit_code
            [[ -z "$learner_exit_code" ]] || echo "learner exit: $learner_exit_code"

            if [[ "$learner_exit_code" == "0" ]]; then
                # update timestamp for PROCESSING (ensure that we store the last timestamp for distributed jobs)
                updateStatusTimestamp PROCESSING $JOB_STATE_DIR/learner.start_time
                # record new status
                recordStatus STORING
                start_lc_wait=`date +%s`
                setState LC_WAIT_ON_SUCCESS
            elif [[ ! -z "$learner_exit_code" ]]; then
                echo "Failed: learner_exit_code: $learner_exit_code" >> $user_log_file
                recordStatus STORING
                start_lc_wait=`date +%s`
                setState LC_WAIT_ON_FAILURE
            elif [[ -f "$halt_file" ]]; then
                echo "halt: learner_exit_code: $learner_exit_code" >> $user_log_file
                start_lc_wait=`date +%s`
                setState LC_WAIT_ON_HALT
            fi
            ;;
        (LC_WAIT_ON_SUCCESS | LC_WAIT_ON_FAILURE | LC_WAIT_ON_HALT)
            end_lc_wait=`date +%s`
            duration_wait=$((end_lc_wait-start_lc_wait))
            if [[ -f "$lc_exit_file" ]]; then
                echo "$current_state: log-collector signaled it's done"
                setState ${lc_transitions[$current_state]}
            elif [ ${duration_wait} -gt ${TIME_TO_SLEEP_FOR_LOG_COLLECTOR} ]; then
                echo "$current_state: time out waiting for log collector"
                echo -1 > $lc_exit_file
                setState ${lc_transitions[$current_state]}
            fi
            ;;
       (STORING_ON_SUCCESS)
            # STORING_ON_SUCCESS -> COMPLETED if storing succeeds
            # STORING_ON_SUCCESS -> FAILED if storing fails
            getExitCode store-results; store_results_exit_code=$exit_code
            getExitCode store-logs; store_logs_exit_code=$exit_code
            [[ -z "$store_results_exit_code" ]] || echo "store-results exit: $store_results_exit_code"
            [[ -z "$store_logs_exit_code" ]] || echo "store-logs exit: $store_logs_exit_code"

            if [[ "$store_results_exit_code" == "0" && "$store_logs_exit_code" == "0" ]]; then
                # Successfully stored both results and logs.
                recordStatus COMPLETED
                setState COMPLETED
            elif [[ ! -z "$store_results_exit_code" && ! -z "$store_logs_exit_code" ]]; then
                # Finished storing both logs and results, but at least one of them had an error.
                echo "Failed: store_results_exit_code: $store_results_exit_code, $store_logs_exit_code: $store_logs_exit_code" >> $user_log_file
                recordStatus FAILED S303 $store_results_exit_code
                setState FAILED
            fi
            ;;
        (STORING_ON_FAILURE)
            # STORING_ON_FAILURE -> FAILED always
            getExitCode store-results; store_results_exit_code=$exit_code
            getExitCode store-logs; store_logs_exit_code=$exit_code
            [[ -z "$store_results_exit_code" ]] || echo "store-results exit: $store_results_exit_code"
            [[ -z "$store_logs_exit_code" ]] || echo "store-logs exit: $store_logs_exit_code"

            if [[ ! -z "$store_results_exit_code" && ! -z "$store_logs_exit_code" ]]; then
                # Finished storing both logs and results. Zero, one, or both of them may have an error, but we don't care.
                echo "Failed: store_results_exit_code: $store_results_exit_code, $store_logs_exit_code: $store_logs_exit_code" >> $user_log_file
                sleep ${TIME_TO_SLEEP_AFTER_USER_LOG}
                # Set status to FAILED and report client error code (see jobmonitor.go for error codes)
                recordStatus FAILED C201 $learner_exit_code
                setState FAILED
            fi
            ;;
        (STORING_ON_HALTED)
            # STORING_ON_HALTED -> HALTED if storing succeeds
            # STORING_ON_HALTED -> FAILED if storing fails
            getExitCode store-results; store_results_exit_code=$exit_code
            getExitCode store-logs; store_logs_exit_code=$exit_code
            [[ -z "$store_results_exit_code" ]] || echo "store-results exit: $store_results_exit_code"
            [[ -z "$store_logs_exit_code" ]] || echo "store-logs exit: $store_logs_exit_code"

            if [[ "$store_results_exit_code" == "0" && "$store_logs_exit_code" == "0" ]]; then
                # Successfully stored both results and logs.
                recordStatus HALTED
                setState HALTED
            elif [[ ! -z "$store_results_exit_code" && ! -z "$store_logs_exit_code" ]]; then
                # Finished storing both logs and results, but at least one of them had an error.
                echo "Failed: store_results_exit_code: $store_results_exit_code, $store_logs_exit_code: $store_logs_exit_code" >> $user_log_file
                recordStatus FAILED S305 $store_results_exit_code
                setState FAILED
            fi
            ;;
        (COMPLETED)
            cleanup
            setState FINAL
            ;;
        (FAILED)
            cleanup
            setState FINAL
            ;;
        (HALTED)
            cleanup
            setState FINAL
            ;;
        (FINAL)
            ;;
        (*)
            echo ERROR: In unexpected state: $current_state
    esac

    sleep 2
done
