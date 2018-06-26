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

from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import sys
import time

from . import push_em_line
from . import connect


def collect_and_send(em_file: str, should_connect: bool=True):
    if should_connect:
        try:
            print("Trying to connect to Training Data Service (emetrics)")
            sys.stdout.flush()
            tdClient = connect.get_connection()
            if tdClient is not None:
                print("Have connection to Training Data Service (emetrics)")
                sys.stdout.flush()
        except Exception as inst:
            print("Unexpected error when attempting to process evaluation metric record  (emetrics):",
                  sys.exc_info()[0])
            print(inst)
            sys.stdout.flush()
    else:
        print("Not connecting to Training Data Service (emetrics)")
        sys.stdout.flush()
        tdClient = None

    log_line_index = 1
    while not os.path.exists(em_file):
        time.sleep(1)

    # TODO: Keep file_pos stored in file, in case of this container's restart
    file_pos = 0
    while True:
        with open(em_file, 'r') as em_stream:
            try:
                em_stream.seek(file_pos)
                for line in iter(em_stream):
                    log_line_index = push_em_line.push (tdClient, line, "", log_line_index)
            except Exception as inst:
                print("Unexpected error:", str(inst))
                sys.stdout.flush()

            file_pos = em_stream.tell()


