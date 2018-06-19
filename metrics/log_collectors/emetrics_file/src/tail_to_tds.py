#!/usr/bin/env python
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
import argparse

from log_collectors.training_data_service_client import tail_to_tds as tail


def main():
    job_directory = os.environ["JOB_STATE_DIR"]
    log_directory = job_directory + "/logs"
    log_file = job_directory + "/latest-log"

    parser = argparse.ArgumentParser()

    parser.add_argument('--log_file', type=str, default=log_file,
                        help='Log file')

    FLAGS, unparsed = parser.parse_known_args()

    tail.collect_and_send(FLAGS.log_file, True)


if __name__ == '__main__':
    main()
