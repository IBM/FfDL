#!/usr/bin/env python
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


