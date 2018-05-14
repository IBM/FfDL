#!/usr/bin/env python

import time
import threading
import logging
import grpc
import os
from typing import List

from log_collectors.training_data_service_client import print_json

from log_collectors.training_data_service_client import training_data_pb2_grpc as td
from log_collectors.training_data_service_client import training_data_pb2 as tdp


class TrainingDataClientBuffered(object):
    """Buffer the TDS events, and flush them after a certain amount of time
    has passed, or when the buffer reaches a certain size.

    This class will also write the emetrics file, thus it is required to have
    and instance specific to the job or sub-job being monitored.

    The instances of this class can me accessed from multiple threads, so it
    must be very threadsafe.  All public methods are blocking.
    """

    CONST_FLUSH_NUMBER_RECORDS = 10
    CONST_FLUSH_TIME = 2.0

    GRPC_RETRY_MAX = 6
    GRPC_RETRY_SLEEP_INTERVAL = 0.5

    BUFFER_MAX_TIL_DUMP = 1024

    emetrics_buf = []   # type: List[tdp.EMetrics]
    emetrics_buf_start_add_time = 0  # type: float

    logline_buf = []   # type: List[tdp.LogLine]
    logline_buf_start_add_time = 0  # type: float

    buffer_lock = threading.Lock()
    other_tdcb_instances = []   # type: List[TrainingDataClientBuffered]

    def __init__(self, td_client: td.TrainingDataStub, em_file_path: str=None):
        """Constructor.

        Args:
          td_client: A td.TrainingDataStub
          em_file_path: If non-null, path where the metrics will be written
        """

        self.td_client = td_client
        self.em_file_path = em_file_path

        # Buffer for what should go into this instances emetrics file
        self.emetrics_file_buf = []   # type: tdp.EMetrics

        self.logger = logging.getLogger("tds_client_buffered")
        self.logger.setLevel(logging.INFO)
        self.logger.info("Creating TrainingDataClientBuffered")

        self.last_em_file_size = 0

        with TrainingDataClientBuffered.buffer_lock:
            if em_file_path is not None:
                for tdcb in TrainingDataClientBuffered.other_tdcb_instances:
                    if tdcb.em_file_path is not None:
                        self.logger.error("only ONE EM writer per process")
                        raise Exception("only ONE EM writer per process")
            TrainingDataClientBuffered.other_tdcb_instances.append(self)

    def set_em_file_path(self, em_file_path: str):
        # Locks are probably overkill, but ensures we're safe
        with TrainingDataClientBuffered.buffer_lock:
            self.logger.info("Setting the em file path: %s", em_file_path)
            self.em_file_path = em_file_path

    def __should_flush(self, buf: [], buf_start_add_time: float):
        return len(buf) >= self.CONST_FLUSH_NUMBER_RECORDS or \
               (time.time() - buf_start_add_time) > self.CONST_FLUSH_TIME

    def __write_em_file_path(self, force: bool=False):
        # The force flag is sent by the caller, which is sent by someone else,
        # which signals intent.  Even though it's not being used right now,
        # I'd rather keep the argument for the moment.
        del force

        if self.em_file_path is not None and len(self.emetrics_file_buf):
            lines_written = 0
            try:
                self.logger.info("writing %d records to %s, thread index %d",
                                  len(self.emetrics_file_buf), self.em_file_path, threading.get_ident())

                # The lines below with "Flush nfs buffer??" are about trying to make sure the nfs cache is flushed.
                # The result of a long struggle, there's probably a better way.

                if self.last_em_file_size > 0:
                    # Flush nfs buffer??
                    if not os.path.exists(self.em_file_path):
                        self.logger.error("file was created, but now it doesn't exist!!! %s", self.em_file_path)

                with open(file=self.em_file_path,  mode='a', buffering=-1) as em_stream:
                    for emetrics in self.emetrics_file_buf:
                        try:
                            json_form = print_json.to_string(emetrics)
                            em_stream.write(json_form)
                            em_stream.write("\n")
                            lines_written += 1
                        except OSError as err:
                            self.logger.warning("Unexpected error writing emetrics file: %s", err)

                # Please keep this in place for now for debugging.
                # # if force:
                # Flush nfs buffer??
                fd = os.open(self.em_file_path, os.O_RDONLY)
                after_stat = os.fstat(fd)
                os.close(fd)

                if after_stat.st_size <= self.last_em_file_size:
                    self.logger.error("what?: file grew smaller! b: %d, a: %d",
                                      self.last_em_file_size, after_stat.st_size)

                self.last_em_file_size = after_stat.st_size

            except OSError as error:  # parent of IOError, OSError *and* WindowsError where available
                self.logger.warning("Unexpected error opening emetrics file: %s", error)
            finally:
                self.emetrics_file_buf = self.emetrics_file_buf[lines_written:]

    def __add_emetrics_with_retry(self, force: bool=False)->bool:
        success = False
        for retryCount in range(0, self.GRPC_RETRY_MAX):
            try:
                self.logger.debug("calling AddEMetricsBatch, adding %d records",
                                  len(TrainingDataClientBuffered.emetrics_buf))
                self.td_client.AddEMetricsBatch(tdp.EMetricsBatch(force=force,
                                                                  emetrics=TrainingDataClientBuffered.emetrics_buf))
                success = True
            except grpc.RpcError as rpc_error:
                self.logger.warning("RpcError error sending emetrics: %s", rpc_error)
            finally:
                TrainingDataClientBuffered.emetrics_buf = []
                TrainingDataClientBuffered.emetrics_buf_start_add_time = time.time()
            if success:
                break
            else:
                time.sleep(self.GRPC_RETRY_SLEEP_INTERVAL)
        return success

    def __add_loglines_with_retry(self, force: bool=False)->bool:
        success = False
        for retryCount in range(0, self.GRPC_RETRY_MAX):
            try:
                self.logger.debug("calling AddLogLineBatch, adding %d lines",
                                  len(TrainingDataClientBuffered.logline_buf))

                self.td_client.AddLogLineBatch(
                    tdp.LogLineBatch(force=force, logLine=TrainingDataClientBuffered.logline_buf))
                success = True
            except grpc.RpcError as rpc_error:
                self.logger.warning("RpcError error sending log lines: %s", rpc_error)
            finally:
                TrainingDataClientBuffered.logline_buf = []
                TrainingDataClientBuffered.logline_buf_start_add_time = time.time()
            if success:
                break
            else:
                time.sleep(self.GRPC_RETRY_SLEEP_INTERVAL)
        return success

    def __flush_emetrics_maybe(self, force: bool=False)->bool:
        success = False
        if ((force and len(TrainingDataClientBuffered.emetrics_buf) > 0) or
                (self.__should_flush(TrainingDataClientBuffered.emetrics_buf,
                                     TrainingDataClientBuffered.emetrics_buf_start_add_time) and
                 len(TrainingDataClientBuffered.emetrics_buf) > 0)):

            self.__write_em_file_path(force=force)

            success = self.__add_emetrics_with_retry(force=force)
            if len(TrainingDataClientBuffered.emetrics_buf) > self.BUFFER_MAX_TIL_DUMP:
                TrainingDataClientBuffered.emetrics_buf = []
                TrainingDataClientBuffered.emetrics_buf_start_add_time = time.time()

        return success

    def __flush_loglines_maybe(self, force: bool=False)->bool:
        success = False
        if (force and len(TrainingDataClientBuffered.logline_buf) > 0) or \
                (self.__should_flush(TrainingDataClientBuffered.logline_buf,
                                     TrainingDataClientBuffered.logline_buf_start_add_time) and
                 len(TrainingDataClientBuffered.logline_buf) > 0):
            success = self.__add_loglines_with_retry(force=force)
            if len(TrainingDataClientBuffered.logline_buf) > self.BUFFER_MAX_TIL_DUMP:
                TrainingDataClientBuffered.logline_buf = []
                TrainingDataClientBuffered.logline_buf_start_add_time = time.time()
        return success

    def __flush_maybe(self, force: bool=False):
        for tdcb in TrainingDataClientBuffered.other_tdcb_instances:
            if tdcb is not self:
                tdcb.__flush_loglines_maybe(force=force)
                tdcb.__flush_emetrics_maybe(force=force)
        self.__flush_loglines_maybe(force=force)
        self.__flush_emetrics_maybe(force=force)

    def flush_maybe(self, force: bool=False):
        with TrainingDataClientBuffered.buffer_lock:
            self.__flush_maybe(force)

    def AddEMetrics(self, emetric_record: tdp.EMetrics):
        with TrainingDataClientBuffered.buffer_lock:
            if len(TrainingDataClientBuffered.emetrics_buf) == 0:
                TrainingDataClientBuffered.emetrics_buf_start_add_time = time.time()
            TrainingDataClientBuffered.emetrics_buf.append(emetric_record)
            # Also append to buffer intended for emetrics file
            if self.em_file_path is not None:
                self.emetrics_file_buf.append(emetric_record)
            self.logger.debug("AddEMetrics: size after add: %d", len(TrainingDataClientBuffered.emetrics_buf))
            self.__flush_maybe()

    def AddLogLine(self, logline_record: tdp.LogLine):
        with TrainingDataClientBuffered.buffer_lock:
            if len(TrainingDataClientBuffered.logline_buf) == 0:
                TrainingDataClientBuffered.logline_buf_start_add_time = time.time()
            TrainingDataClientBuffered.logline_buf.append(logline_record)
            self.logger.debug("AddLogLine: size after add: %d", len(TrainingDataClientBuffered.logline_buf))
            self.__flush_maybe()
