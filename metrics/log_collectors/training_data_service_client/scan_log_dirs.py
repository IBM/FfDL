#!/usr/bin/env python
"""Scan files and directories for logs and emetrics"""

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
import time
import sys
import logging
import threading

from typing import Dict, Callable, Any, Tuple

from log_collectors.training_data_service_client import extract_datetime as extract_datetime
from log_collectors.training_data_service_client import training_data_buffered as tdb

from . import push_log_line
from . import connect
from . import states
from . import match_log_file


class LinesCounter:

    def __init__(self, start=0):
        self.lock = threading.Lock()
        self.value = start
        self.time_of_last_count = None

    def increment(self):
        # logging.debug('Waiting for lock')
        self.lock.acquire()
        try:
            # logging.debug('Acquired lock')
            self.value = self.value + 1
            self.time_of_last_count = time.time()
        finally:
            self.lock.release()

    def add(self, count: int):
        # logging.debug('Waiting for lock')
        self.lock.acquire()
        try:
            # logging.debug('Acquired lock')
            self.value = self.value + count
            self.time_of_last_count = time.time()
        finally:
            self.lock.release()

    def time_since_last_entry(self):
        if self.time_of_last_count is not None:
            elapsed_time = time.time() - self.time_of_last_count
            return elapsed_time
        else:
            return -0.0


def stat_nfs_safe_modification_time(file: str)->float:
    # os.stat seems to be completely unreliable for the nfs volume,
    # because file attributes might not have been flushed.
    # By opening the file, it will flush the attributes.
    # Python programming contest time:
    #      what's a faster way to implement this?
    file_fd = os.open( file, os.O_RDONLY )
    stat = os.fstat(file_fd)
    os.close(file_fd)
    # stat = os.stat(self.log_file)

    return stat.st_mtime


class GetAndPushLogLinesRunner(threading.Thread):

    def __init__(self, td_client: tdb.TrainingDataClientBuffered, logfile_year, log_file: str, subdir: str,
                 lines_counter: LinesCounter,
                 extra: Any = None,
                 push_function: Callable[
                     [tdb.TrainingDataClientBuffered, str, str, str, int, int, str, Any],
                     Tuple[int, int]]=push_log_line.push,
                 logger: logging.Logger = logging.getLogger()):
        super().__init__()
        self.td_client = td_client
        self.logfile_year = logfile_year
        self.log_file = log_file
        self.subdir = subdir
        self.lines_counter = lines_counter
        self.push_function = push_function
        self.extra = extra

        self.log_line_index = 1
        self.record_index = 1
        self.logger = logger

        log_dir = os.environ["LOG_DIR"]
        path_relative_log_dir = self.log_file[len(log_dir):]

        unique_id = path_relative_log_dir.replace(os.path.sep, "_")
        file_pos_file_base = ".lc_pos_info_"+unique_id

        job_state_dir = os.environ["JOB_STATE_DIR"]
        self.file_pos_file = os.path.join(job_state_dir, file_pos_file_base)

        self.logger.debug("file_pos_file: %s", self.file_pos_file)

    def processLogLines(self, file_pos: int) -> int:
        # if os.path.exists(self.file_pos_file):
        #     with open(self.file_pos_file, "r") as file_pos_file_stream:
        #         file_pos, log_line_index, record_index = [int(x) for x in next(file_pos_file_stream).split()]
        #         self.logger.debug("read pos: %s %s %s" % (file_pos, log_line_index, record_index))
        with open(self.log_file, 'r') as log_stream:
            try:
                # self.logger.debug("GetAndPushLogLines seeking to: %d" % file_pos)
                log_stream.seek(file_pos)
                for line in iter(log_stream):
                    # self.logger.debug("Pushing logline: %s", line)
                    self.log_line_index, self.record_index = \
                        self.push_function(self.td_client, self.log_file, line,
                                           self.logfile_year,
                                           self.log_line_index,
                                           self.record_index,
                                           self.subdir, self.extra)
                    self.lines_counter.increment()
                    time.sleep(0)
            except Exception as inst:
                self.logger.exception("Exception")
                self.logger.error("Unexpected error: %s, %s", str(inst), self.log_file)
            finally:
                file_pos = log_stream.tell()
                # self.logger.debug("GetAndPushLogLines tell pos: %d" % file_pos)
                # with open(self.file_pos_file, "w") as file_pos_file_stream:
                #     self.logger.debug("write pos: %s %s %s" % (file_pos, log_line_index, record_index))
                #     file_pos_file_stream.write("%s %s %s" % (file_pos, log_line_index, record_index))

        return file_pos

    def run(self):
        self.logger.info("GetAndPushLogLinesRunner thread running: %s" % self.log_file)

        file_pos = 0
        last_modified_time = 0.0
        shutdown_start_time = 0.0

        states.register_scanner()

        try:
            os.stat_float_times(True)
            while True:
                # elapsed_time_since_last_mod = time.time() - last_modified_time
                # self.logger.debug("time_since_last_push %r", time_since_last_push)
                # elapsed_time_since_last_mod > 2.0 and
                if shutdown_start_time == 0.0 and states.is_learner_done(logger=self.logger):
                    self.logger.info("Learner done, begin GetAndPushLogLinesRunner shutdown: %s", self.log_file)
                    shutdown_start_time = time.time()

                if shutdown_start_time != 0.0:
                    elapsed_time_since_shutdown = (time.time() - shutdown_start_time)
                    if elapsed_time_since_shutdown > states.DURATION_SHUTDOWN_DELAY:
                        self.logger.info("time since shutdown start: %f, Flushing GetAndPushLogLinesRunner with force: %s",
                                     elapsed_time_since_shutdown, self.log_file)

                        self.td_client.flush_maybe(force=True)

                        time.sleep(states.SLEEP_BEFORE_LC_DONE)

                        break

                try:
                    latest_modified_time = stat_nfs_safe_modification_time(self.log_file)

                    if last_modified_time == latest_modified_time:
                        self.logger.debug("file mod NOT changed: %f, %s", latest_modified_time, self.log_file)
                        self.td_client.flush_maybe()
                        time.sleep(.25)  # Micro sleep
                        continue
                    else:
                        self.logger.debug("file mod changed: %f, %s", latest_modified_time, self.log_file)
                        last_modified_time = latest_modified_time
                except OSError as oserr:
                    # Doesn't exist?
                    self.logger.info("OSError: %s", str(oserr))
                    time.sleep(0.25)  # Micro sleep
                    continue

                file_pos = self.processLogLines(file_pos)
                self.td_client.flush_maybe()

        finally:
            self.logger.info("Signaling GetAndPushLogLinesRunner is done: %s", self.log_file)
            # if not self.running_in_foreground:
            #     # If we're running in background, assume the main thread will do the signal
            #     time.sleep(15)
            states.signal_lc_done(logger=self.logger)
            # I seem to have problems exiting directly, so, this sleep seems to help.
            # My unsubstantiated theory is that gRPC needs time to flush.
            # Note since we signaled, we won't actually wait n seconds, the
            # job monitor will delete us.
            time.sleep(states.SLEEP_BEFORE_EXIT_TIME)
            self.logger.info("Exiting GetAndPushLogLinesRunner: %s", self.log_file)


class LogScanner:

    def __init__(self, should_connect: bool=True, logger: logging.Logger = logging.getLogger()):

        self.lines_counter = LinesCounter()  # type: LinesCounter
        self.log_runners = {}                # type: Dict[str, threading.Thread]
        self.td_client = None                # type: tdb.TrainingDataClientBuffered
        self.logger = logger

        if should_connect:
            try:
                self.logger.debug("tail_to_tds: Trying to connect to Training Data Service (log lines)")
                self.td_client = tdb.TrainingDataClientBuffered(connect.get_connection())
                if self.td_client.td_client is not None:
                    self.logger.debug("Have connection to Training Data Service (log lines)")
            except Exception as inst:
                self.logger.error("Unexpected error when attempting to connect: %r" +
                              sys.exc_info()[0])
                self.logger.debug(inst)

    def scan(self, log_dir: str,
             extra: Any = None,
             is_log: Callable[[str], bool] = match_log_file.is_log_file,
             push_function: Callable[
                 [tdb.TrainingDataClientBuffered, str, str, str, int, int, str, Any],
                 Tuple[int, int]]=push_log_line.push,
             is_emetrics: Callable[[str], bool] = None,
             push_emetrics_function: Callable[
                 [tdb.TrainingDataClientBuffered, str, str, str, int, int, str, Any],
                 Tuple[int, int]]=None,
             is_subdir_function: Callable[[str], bool]=os.path.isdir,
             should_loop: bool=True):

        while True:
            # The design allows for there to be top-level log files, as well as subdirectory logs that
            # contain some sub-run logs.  These are indexed in the training data service with the subdir
            # in the MetaInfo inner struct.
            # self.logger.debug("tail_to_tds: does path exist? %s" % log_dir)
            if not os.path.exists(log_dir):
                time.sleep(.5)
                continue

            # self.logger.debug("tail_to_tds: path does exist: %s" % log_dir)
            try:
                logfile_year = extract_datetime.get_log_created_year(log_dir)
                for file_or_dir_name in os.listdir(log_dir):
                    sub_dir_path = os.path.join(log_dir, file_or_dir_name)
                    # self.logger.debug("considering if %s is a directory", sub_dir_path)
                    if is_subdir_function(sub_dir_path):
                        sub_dir_id = file_or_dir_name
                        # Now iterate over the files in the subdirectory, hoping to find a logfile
                        for file_in_subdir in os.listdir(sub_dir_path):
                            log_file_path = os.path.join(sub_dir_path, file_in_subdir)
                            if log_file_path not in self.log_runners:
                                push_function_arg = None
                                if is_log is not None and is_log(os.path.basename(log_file_path)):
                                    push_function_arg = push_function
                                elif is_emetrics is not None and is_emetrics(os.path.basename(log_file_path)):
                                    push_function_arg = push_emetrics_function

                                if push_function_arg is not None:
                                    self.logger.info("Creating sub-level log runner for %s", log_file_path)
                                    runner = GetAndPushLogLinesRunner(
                                        td_client=tdb.TrainingDataClientBuffered(self.td_client.td_client),
                                        logfile_year=logfile_year,
                                        log_file=log_file_path,
                                        subdir=sub_dir_id,
                                        lines_counter=self.lines_counter,
                                        extra=extra,
                                        push_function=push_function_arg,
                                        logger=self.logger)
                                    self.log_runners[log_file_path] = runner
                                    runner.start()
                    else:
                        log_file_path = os.path.join(log_dir, file_or_dir_name)

                        if log_file_path not in self.log_runners:
                            push_function_arg = None

                            if is_log is not None and is_log(file_or_dir_name):
                                push_function_arg = push_function
                            elif is_emetrics is not None and is_emetrics(file_or_dir_name):
                                push_function_arg = push_emetrics_function

                            if push_function_arg is not None:
                                self.logger.info("Creating top-level log runner for %s", log_file_path)
                                runner = GetAndPushLogLinesRunner(
                                    td_client=self.td_client,
                                    logfile_year=logfile_year,
                                    log_file=log_file_path,
                                    subdir="",
                                    lines_counter=self.lines_counter,
                                    extra=extra,
                                    push_function=push_function_arg,
                                    logger=self.logger)
                                self.log_runners[log_file_path] = runner
                                runner.start()

                time.sleep(.5)

            except Exception as inst:
                self.logger.error("Error thrown (recovering): %r", sys.exc_info()[0])

            if should_loop is False:
                break
