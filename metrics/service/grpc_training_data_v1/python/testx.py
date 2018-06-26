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

import os
import re
import time
import datetime
import sys
import extract_seconds

import json

from google.protobuf import json_format

import training_data_service_client.training_data_pb2 as tdp
# from .training_data_service_client import training_data_pb2_grpc as td
# from training_data_service_client import connect
# from training_data_service_client import push_log_line

# import parse_log_incremental
def totimestamp(dt, epoch=datetime.datetime(1970,1,1)):
    td = dt - epoch
    # return td.total_seconds()
    return int((td.microseconds + (td.seconds + td.days * 86400) * 10**6) / 10**6)

try:
    rowdict = dict()
    rowdict['NumIters'] = '5'
    rowdict['Seconds'] = 34

    rowdict['LearningRate'] = '2.43'
    rowdict['loss'] = '7.82'

    etimes = dict()
    # etimes['iteration'] = i*10
    etimes['iteration'] = tdp.Any(type=tdp.Any.INT, value=str(int(rowdict['NumIters'])))
    d = datetime.datetime.today() + datetime.timedelta(seconds=rowdict['Seconds'])
    etimes['timestamp'] = tdp.Any(type=tdp.Any.STRING, value=d.isoformat("T") + "Z")

    d = datetime.datetime.utcnow()

    timestamp = totimestamp(d)

    valuedict = dict()
    # for k in rowdict.keys():
    #     valuedict[k] = tdp.Any(type=tdp.Any.FLOAT, value=str(rowdict[k]))

    if 'LearningRate' in rowdict:
        valuedict['lr'] = tdp.Any(type=tdp.Any.FLOAT, value=str(rowdict['LearningRate']))
    if 'loss' in rowdict:
        valuedict['loss'] = tdp.Any(type=tdp.Any.FLOAT, value=str(rowdict['loss']))
    if 'accuracy' in rowdict:
        valuedict['accuracy'] = tdp.Any(type=tdp.Any.FLOAT, value=str(rowdict['accuracy']))

    emetrics = tdp.EMetrics(
        meta=tdp.MetaInfo(
            training_id="training-abc",
            time=timestamp,
            rindex=1
        ),
        grouplabel="blah",
        etimes= etimes,
        values=valuedict
    )
    # for k in rowdict.keys():
    #     emetrics.EtimesEntry[k] = tdp.Any(type=tdp.Any.FLOAT, value=str(rowdict[k]))

    # td_client.AddEMetrics(emetrics)

    # We potentially still want to output to stdout in order to get picked up by fluentd or whoever.
    # This doesn't work, at least without a custom deserializer:
    # json_string = json.dumps(emetrics, skipkeys=True, sort_keys=False,
    #                          separators=(',\t', ': '))
    # ...also, the gRPC generated code doesn't generate legal json when marshaling to string.
    # So, hack around this for the moment.

    json_string = json_format.MessageToJson(emetrics)
    # json_string = str(emetrics)
    # json_string = json_string.replace('\n', '').replace('\r', '')
    # json_string = json_string.replace('type: STRING', 'type: 0').replace('type: JSONSTRING', 'type: 1').replace('type: INT', 'type: 2').replace('type: FLOAT', 'type: 3')
    # print(json_string)
    print(json_string)

except Exception as inst:
    print("Unexpected error when attempting to send emetrics:", sys.exc_info()[0])
    print(type(inst))
    print(inst.args)
    print(inst)
    sys.stdout.flush()


