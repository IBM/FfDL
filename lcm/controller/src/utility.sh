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

STATSDEXPORTER_HOST="statsdexporter"
STATSDEXPORTER_UDP_PORT="9125"

# Retries a command a with backoff.
# The retry count is given by ATTEMPTS (default 5), the
# initial backoff timeout is given by TIMEOUT in seconds
# (default 1.)
#
# Successive backoffs double the timeout.
#
# Beware of set -e killing your whole script!
function with_backoff {
  local max_attempts=${ATTEMPTS-5}
  local timeout=${TIMEOUT-1}
  local attempt=0
  local exitCode=0

  while [[ $attempt < $max_attempts ]]
  do
    "$@"
    exitCode=$?

    if [[ $exitCode == 0 ]]
    then
      break
    elif [[ $exitCode == 2 ]]
    then
       attempt=0
       timeout=${TIMEOUT-1}
    fi

    echo "Failure! Retrying in $timeout.." 1>&2
    #assumption is $1 is the command and $2 is the argument to the command
    updateMetricsOnETCDFailure "$1.$2" $attempt &

    sleep $timeout
    attempt=$(( attempt + 1 ))
    timeout=$(( timeout * 2 ))
  done

  if [[ $exitCode != 0 ]]
  then
    echo "You've failed me for the last time! ($@)" 1>&2
  fi

  return $exitCode
}

# Retries a command a with backoff.
# does infinite retries in an exponential manner
# starts with 2 seconds up to 90 seconds and then backsoff again and restarts
function infinite_exp_backoff {
  local max_time=90
  local default_timeout=2
  local timeout=$default_timeout

  while [[ timeout -le max_time ]]
  do
    "$@"
    exitCode=$?

    if [[ $exitCode == 0 ]]
    then
      break
    fi

    echo "Failure! Retrying in $timeout.." 1>&2
    sleep $timeout
    timeout=$(( timeout * 2 ))

    #reset the timeout back to 0
    if [[ timeout -ge  max_time ]]
    then
      timeout=$default_timeout
    fi

  done

  return $exitCode

}

# Run a etcdctl command.
# Args: The arguments to pass to "etcdctl -c ...".
#       For example, pass "create /foo/bat" to run "etcdctl set /foo/bar"
# Returns non-zero value if the etcdctl command has any stderr output.
# This function is needed because the etcdctl command returns 0 even on failure.
function runEtcdCommand() {

  # Run command.
  echo "etcdctl args: $@"

  cert_file_path=/etc/certs/etcd/etcd.cert

  ETCDCTL_API=3 etcdctl --user=${DLAAS_ETCD_USERNAME}:${DLAAS_ETCD_PASSWORD} --insecure-skip-tls-verify=true --cacert ${cert_file_path} --dial-timeout=10s --endpoints $DLAAS_ETCD_ADDRESS "$@" 2> /tmp/stderr

  exitcode=$?
  echo "etcdctl exitcode: $exitcode"

  # Return non-zero exit code if there's any stderr output.
  if [ -s "/tmp/stderr" ]; then
    cat /tmp/stderr
    return 1
  fi

  return $exitcode
}

# This function should be run in background using &
# Args:  reason for failure 
# Limitation with this function for now is that it starts a new time series per training failure
# TODO use etcd to store value of the current failure counter
function updateMetricsOnTrainingFailure() {
  status=$1
  metrics="controller.training.failures.$status:1|c"
  echo "got metrics to push as $metrics"
  pushMetrics $metrics

}

# This function should be run in background using &
# Args:  operation , counter 
# Limitation with this function for now is that it starts a new time series per training failure
# TODO use etcd to store value of the current failure counter
function updateMetricsOnETCDFailure() {
  operation=$1
  counter=$2
  metrics="controller.etcd.failures.$operation.$counter:1|c"
  echo "got metrics to push as $metrics"
  pushMetrics $metrics
}

function pushMetrics() {
  metrics=$1
  # Setup UDP socket with statsd server
  exec 3<> /dev/udp/$STATSDEXPORTER_HOST/$STATSDEXPORTER_UDP_PORT
  # Send data
  printf "$metrics" >&3
  # Close UDP socket
  exec 3<&-
  exec 3>&-
}
