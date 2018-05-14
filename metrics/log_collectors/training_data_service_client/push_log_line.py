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

import os
import logging

from typing import Tuple, Any

from log_collectors.training_data_service_client import print_json

from log_collectors.training_data_service_client import training_data_buffered as tdb
from log_collectors.training_data_service_client import training_data_pb2 as tdp
from log_collectors.training_data_service_client import extract_datetime as edt


def push(td_client: tdb.TrainingDataClientBuffered, log_file: str, log_line: str, logfile_year: str, rindex: int, rindex2: int,
         subdir: str, extra: Any=None) -> Tuple[int, int]:
    """Push the processed metrics data to the metrics service"""

    del log_file  # Ignored parameter for now
    del logfile_year  # Ignored parameter for now
    del extra  # Ignored parameter for now

    logging.debug("Creating logline record: %d", rindex)
    # logging.debug("Sending %d (%s) %s", rindex, subdir, log_line)
    log_line_record = tdp.LogLine(
        meta=tdp.MetaInfo(
            training_id=os.environ["TRAINING_ID"],
            time=edt.get_meta_timestamp(),
            rindex=rindex,
            subid=subdir
        ),
        line=log_line
    )

    if td_client is not None:
        logging.debug("Calling AddLogLine: %d", rindex)
        td_client.AddLogLine(log_line_record)

    return rindex+1, rindex2
