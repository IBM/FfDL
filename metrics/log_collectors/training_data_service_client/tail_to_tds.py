#!/usr/bin/env python
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import os
import time
import sys

from log_collectors.training_data_service_client import extract_datetime as extract_datetime

from . import push_log_line
from . import connect


def collect_and_send(log_file: str, should_connect: bool=True):
    if should_connect:
        try:
            print("Trying to connect to Training Data Service (log lines)")
            sys.stdout.flush()
            tdClient = connect.get_connection()
            if tdClient is not None:
                print("Have connection to Training Data Service (log lines)")
                sys.stdout.flush()
        except Exception as inst:
            print("Unexpected error when attempting to process evaluation metric record  (log lines):",
                  sys.exc_info()[0])
            print(inst)
            sys.stdout.flush()

    log_line_index = 1

    while not os.path.exists(log_file):
        time.sleep(1)

    logfile_year = extract_datetime.get_log_created_year(log_file)

    # TODO: Keep file_pos stored in file, in case of this container's restart
    file_pos = 0
    while True:
        with open(log_file, 'r') as em_stream:
            try:
                em_stream.seek(file_pos)
                for line in iter(em_stream):
                    log_line_index = push_log_line.push (tdClient, line, logfile_year, log_line_index)
            except Exception as inst:
                print("Unexpected error:", str(inst))
                sys.stdout.flush()
    
            file_pos = em_stream.tell()
