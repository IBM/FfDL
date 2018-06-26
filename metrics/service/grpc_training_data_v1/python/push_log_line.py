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
import datetime
import time

if sys.version_info[0] < 3:
    from training_data_service_client import training_data_pb2_grpc as td
    from training_data_service_client import training_data_pb2 as tdp
else:
    from .training_data_service_client import training_data_pb2_grpc as td
    from .training_data_service_client import training_data_pb2 as tdp

def totimestamp(dt, epoch=datetime.datetime(1970,1,1)):
    td = dt - epoch
    # return td.total_seconds()
    return int((td.microseconds + (td.seconds + td.days * 86400) * 10**6) / 10**6)

def extract_datetime_from_line(line, year):
    # Expected format: I0210 13:39:22.381027 25210 solver.cpp:204] Iteration 100, lr = 0.00992565
    line = line.strip().split()
    month = int(line[0][1:3])
    day = int(line[0][3:])
    timestamp = line[1]
    pos = timestamp.rfind('.')
    ts = [int(x) for x in timestamp[:pos].split(':')]
    hour = ts[0]
    minute = ts[1]
    second = ts[2]
    microsecond = int(timestamp[pos + 1:])
    dt = datetime.datetime(year, month, day, hour, minute, second, microsecond)
    return dt

def push_logline(td_client, logline, logfile_year, rindex):
    '''Push the processed metrics data to the metrics service'''

    # ms = time.time()
    try:
        line_time = \
            extract_datetime_from_line(
                logline, logfile_year)
        ms = totimestamp(line_time)
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

        td_client.AddLogLine(logLineRecord)

    except Exception as inst:
        print("Unexpected error when attempting to send logline:", sys.exc_info()[0])
        print(type(inst))
        print(inst.args)
        print(inst)
        sys.stdout.flush()

    # json_string = str(logLineRecord)
    # json_string = json_string.replace('\n', '').replace('\r', '')
    # # json_string = json_string.replace('type: STRING', 'type: 0').replace('type: JSONSTRING', 'type: 1').replace('type: INT', 'type: 2').replace('type: FLOAT', 'type: 3')
    # print(json_string)

    return rindex+1
