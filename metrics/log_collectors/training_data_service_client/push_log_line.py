#!/usr/bin/env python
"""Push log line to data service or somewhere"""

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

import sys
import os
import time

# from log_collectors.training_data_service_client import training_data_pb2_grpc as td
from log_collectors.training_data_service_client import training_data_pb2 as tdp

from log_collectors.training_data_service_client import extract_datetime as extract_datetime
from log_collectors.training_data_service_client import print_json as print_json


def push(td_client, logline, logfile_year, rindex):
    """Push the processed metrics data to the metrics service"""

    # ms = time.time()
    try:
        line_time, _ = \
            extract_datetime.extract_datetime(
                logline, logfile_year)
        ms = extract_datetime.to_timestamp(line_time)
    except ValueError:
        ms = time.time()

    try:

        logLineRecord = tdp.LogLine(
            meta=tdp.MetaInfo(
                training_id=os.environ["TRAINING_ID"],
                time=int(ms),
                rindex=rindex
            ),
            line=logline
        )
        if td_client is not None:
            td_client.AddLogLine(logLineRecord)

        # Uncomment for debugging
        # print_json.output(logLineRecord)

    except Exception as inst:
        print("Unexpected error when attempting to send logline:", sys.exc_info()[0])
        print(type(inst))
        print(inst.args)
        print(inst)
        sys.stdout.flush()

    return rindex+1
