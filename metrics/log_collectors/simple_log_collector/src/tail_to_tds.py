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
import logging

from log_collectors.training_data_service_client import match_log_file
from log_collectors.training_data_service_client import push_log_line
from log_collectors.training_data_service_client import scan_log_dirs


def main():
    logging.basicConfig(format='%(filename)s %(funcName)s %(lineno)d: %(message)s', level=logging.INFO)
    log_directory = os.environ["LOG_DIR"]
    # log_file = log_directory + "/latest-log"

    parser = argparse.ArgumentParser()

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='Log directory')

    FLAGS, unparsed = parser.parse_known_args()

    scan_log_dirs.LogScanner(should_connect=True).scan(
        log_dir=FLAGS.log_dir,
        is_log=match_log_file.is_log_file,
        push_function=push_log_line.push)


if __name__ == '__main__':
    main()
