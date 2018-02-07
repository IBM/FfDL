#!/usr/bin/env python
"""Push evaluation metrics record json line to data service or somewhere"""

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
import json
import os
from log_collectors.training_data_service_client import training_data_pb2 as tdp
from log_collectors.training_data_service_client import print_json as print_json


def push(td_client, logline: str, logfile_year, rindex):
    """Push the processed metrics data to the metrics service"""

    del logfile_year  # Ignored parameter for now
    try:
        emr = json.loads(logline+"\n")

        etimes_dict = emr["etimes"]
        etimes: dict = dict()
        for key, value_any in etimes_dict.items():
            val = value_any["value"]
            if "type" in value_any:
                val_type = value_any["type"]
                etimes[key] = tdp.Any(type=val_type, value=str(val))
            else:
                etimes[key] = tdp.Any(value=str(val))

        values_dict = emr["values"]
        scalars: dict = dict()
        for key in values_dict:
            value_any = values_dict[key]
            val = value_any["value"]
            if "type" in value_any:
                val_type = value_any["type"]
                scalars[key] = tdp.Any(type=val_type, value=val)
            else:
                scalars[key] = tdp.Any(value=val)

        emr_meta = emr["meta"]

        training_id = emr_meta["training_id"]
        if training_id == None and "TRAINING_ID" in os.environ:
            training_id = os.environ["TRAINING_ID"],

        emetrics = tdp.EMetrics(
            meta=tdp.MetaInfo(
                training_id=training_id,
                time=int(emr_meta["time"]),
                rindex=int(emr_meta["rindex"]),
            ),
            grouplabel=emr["grouplabel"],
            etimes=etimes,
            values=scalars,
        )

        if td_client is not None:
            td_client.AddEMetrics(emetrics)

        # for now, print to stdout (support old endpoint).
        # TODO: Don't print to stdout for metrics
        json_form = print_json.to_string(emetrics)
        print(json_form)

    except Exception as inst:
        print("Unexpected error when attempting to process evaluation metric record:", sys.exc_info()[0])
        print(inst)
        sys.stdout.flush()

    return rindex+1
