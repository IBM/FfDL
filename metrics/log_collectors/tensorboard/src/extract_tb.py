#!/usr/bin/env python
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import logging
import time
import threading
from typing import List

import argparse
import sys
import os
import datetime
import traceback

from log_collectors.training_data_service_client import scan_log_dirs
from log_collectors.training_data_service_client import push_log_line

from log_collectors.training_data_service_client import training_data_pb2 as tdp
from log_collectors.training_data_service_client import training_data_buffered as tdb

from log_collectors.training_data_service_client import states

from tensorboard.backend.event_processing import event_accumulator
from tensorboard.backend.event_processing import event_multiplexer

from log_collectors.training_data_service_client import extract_datetime

from log_collectors.training_data_service_client import match_log_file

etb_logger = None   # type: logging.Logger


class Tracker:
    """Representing a group run, such as 'test' or 'train'"""

    def __init__(self,
                 summary_dir: str,
                 log_dir: str,
                 sub_identifier: str,
                 group: str,
                 td_client: tdb.TrainingDataClientBuffered):
        """Return tracker object (meant to be opaque) to be passed to emitEvalMetricViaTracker(...)"""

        # Create a tensorboard event accumulator.
        self.summary_dir = summary_dir
        etb_logger.info("Creating ea for %s", self.summary_dir)
        self.ea = event_accumulator.EventAccumulator(self.summary_dir)

        # ea = event_accumulator.EventAccumulator(summary_dir,
        #     size_guidance={
        #                    event_accumulator.COMPRESSED_HISTOGRAMS: 1,
        #                    event_accumulator.IMAGES: 1,
        #                    event_accumulator.AUDIO: 1,
        #                    event_accumulator.SCALARS: 1000,
        #                    event_accumulator.HISTOGRAMS: 1,
        #                    })

        self.log_dir = log_dir
        self.td_client = td_client    # type: tdb.TrainingDataClientBuffered
        self.group = group
        self.sub_identifier = sub_identifier
        self.step_to_event_list_dict = dict()
        self.step_added = -1
        self.wall_time = -1.0

        etb_logger.info("Done initializing tracker for log_dir: %s, summary_dir: %s, em_file_path: %s",
                        self.log_dir, self.summary_dir, self.td_client.em_file_path)

    def is_queued_for_report(self)->bool:
        return self.step_added != -1

    def queued_wall_time(self)->float:
        return self.wall_time

    # Emit evaluation metrics from a tracker object, for visibility to DLaaS clients.
    def track(self)->bool:

        if self.is_queued_for_report():
            return True

        self.ea.Reload()  # loads events from file

        tags = self.ea.Tags()
        scaler_keys = tags[event_accumulator.SCALARS]

        # Have to do essentially a table rotation. The event accumulator API seems
        # only to allow retrieval per scaler, where we need all the scaler values
        # per step.
        # last_processed = self.last_step_processed
        for scaler_key in scaler_keys:

            scalerEvents = self.ea.Scalars(scaler_key)

            did_add = False
            for event in scalerEvents:
                if self.step_added != -1 and event.step != self.step_added:
                    # after the first step found, only use scaler events that match this step
                    continue

                label_event_list = self.step_to_event_list_dict.get(event.step)
                if label_event_list is None:
                    eventSet = EventSet(scaler_key, event)
                    # etb_logger.debug("   event(%s): scaler_key, %r", scaler_key, event)
                    self.step_to_event_list_dict[event.step] = [eventSet]
                    self.step_added = int(event.step)
                    did_add = True
                    # Just track first occurrence time
                    self.wall_time = event.wall_time
                    eventSet.wall_time = event.wall_time
                elif not isInList(label_event_list, scaler_key, event.step, event.wall_time):
                    eventSet = EventSet(scaler_key, event)
                    # etb_logger.debug("   event(%s): scaler_key, %r", scaler_key, event)
                    self.step_to_event_list_dict[event.step].append(eventSet)
                    did_add = True

                if did_add:
                    # intention: break from this scaler investigation, go to the next scaler
                    break

        return self.is_queued_for_report()

    def report(self, rindex: int)->int:
        if self.is_queued_for_report():
            values_dict = {}
            event_list = self.step_to_event_list_dict[self.step_added]
            rfcRFC3339Formatted = None
            event_wall_time = 0.0

            # Transmogrify the ScalerEvent list into a values dictionary
            for event_set in event_list:
                label = event_set.label
                event = event_set.event  # type: event_accumulator.ScalarEvent
                if event_wall_time == 0.0:
                    event_wall_time = float(event.wall_time)
                    if rfcRFC3339Formatted is None:
                        dt = datetime.datetime.fromtimestamp(event.wall_time)
                        rfcRFC3339Formatted = dt.isoformat("T") + "Z"

                values_dict[label] = tdp.Any(type=tdp.Any.FLOAT, value=str(event.value))

            # etb_logger.debug("Calling emitEvalMetric step=%r: rindex=%r", self.step_added, rindex)
            emitEvalMetric(td_client=self.td_client,
                           group_label=self.group,
                           iterStep=self.step_added,
                           timestamp=rfcRFC3339Formatted,
                           values_dict=values_dict,
                           rindex=rindex,
                           event_wall_time=event_wall_time,
                           sub_identifier=self.sub_identifier)
            rindex += 1
            self.step_added = -1
            self.wall_time = -1.0

        return rindex


class Run_tracker(threading.Thread):
    """Representing a directory that contains a Tensorboard summary directory"""

    # See EventMultiplexer, this class is very similar, I think?  Could try to utilize that class
    # and do away with this class.  I think.

    def __init__(self, td_client: tdb.TrainingDataClientBuffered, log_dir: str, sub_identifier: str,
                 lines_counter: scan_log_dirs.LinesCounter):
        super().__init__()
        global etb_logger
        self.log_dir = log_dir
        self.sub_identifier = sub_identifier
        self.event_trackers = []  # type: List[Tracker]
        self.summary_dirs = []  # list of strings representing group names
        self.td_client = td_client
        self.lines_counter = lines_counter

        self.em_file_path = os.path.join(log_dir, match_log_file.EMETRICS_FILE_BASE_NAME)
        self.td_client.set_em_file_path(self.em_file_path)

        etb_logger.info("new Run_tracker for %s", log_dir)

        self.record_index = 1

    @staticmethod
    def sync_event_files_for_nfs_cache(dir_name: str):
        """Make sure event files are synchronized per the nfs cache"""

        for file_name in os.listdir(dir_name):
            full_path = os.path.join(dir_name, file_name)
            if event_accumulator.IsTensorFlowEventsFile(full_path):
                stat = os.stat(full_path)
                before = stat.st_mtime
                synced_modification_time = scan_log_dirs.stat_nfs_safe_modification_time(full_path)
                if synced_modification_time != before:
                    etb_logger.info("sync file: %s, b: %f, a: %f - %s",
                                    full_path, before, synced_modification_time, "CHANGED")

    def build_event_trackers(self):

        # Check to see if new group sub directories have been added
        for summary_dir in event_multiplexer.GetLogdirSubdirectories(self.log_dir + '/'):
            # in the case of this the top level run tracker, this will screen out
            # sub run directories.  Otherwise log_dir will be the sub run directory itself,
            # and this should not fire.
            if not is_sub_run_dir(summary_dir, self.log_dir):
                Run_tracker.sync_event_files_for_nfs_cache(summary_dir)
                group = os.path.basename(summary_dir)
                if summary_dir not in self.summary_dirs:
                    self.summary_dirs.append(summary_dir)
                    tracker = Tracker(summary_dir, self.log_dir, self.sub_identifier, group,
                                      self.td_client)
                    self.event_trackers.append(tracker)

    def track_and_report_one(self, report_only_if_queue_full=False)->int:
        """Keep the event trackers, which point to different event files, queued so the event times
        can be examined, and then report the earliest event."""
        did_report = False
        number_queued = 0
        for tracker in self.event_trackers:
            if tracker.track():
                number_queued += 1

        if report_only_if_queue_full is False or number_queued == len(self.event_trackers):
            earliest_wall_time = sys.float_info.max
            earliest_tracker = None  # type: Tracker
            for i, tracker in enumerate(self.event_trackers):
                if tracker.is_queued_for_report():
                    if tracker.wall_time < earliest_wall_time:
                        earliest_wall_time = tracker.wall_time
                        earliest_tracker = tracker

            if earliest_tracker is not None:
                self.record_index = earliest_tracker.report(self.record_index)
                self.lines_counter.increment()
                did_report = True

        return did_report

    def run(self):
        states.register_scanner()
        shutdown_start_time = 0.0
        time_since_last_learner_done_check = time.time()
        try:
            while True:
                self.build_event_trackers()

                count_main_loop_report = 0
                while self.track_and_report_one(report_only_if_queue_full=True):
                    # loop until track_and_report_one says it didn't report anything
                    count_main_loop_report += 1
                if count_main_loop_report > 0:
                    etb_logger.info("Main loop report, n records: %d", count_main_loop_report)

                if shutdown_start_time == 0.0:
                    # compare the time elapsed since the last report
                    elapsed_since_last_check = time.time() - time_since_last_learner_done_check
                    if elapsed_since_last_check >= 1.0:  # Check every 2.0 seconds
                        if states.is_learner_done(logger=etb_logger):
                            etb_logger.info("Learner done, begin log-collector shutdown: %s", self.log_dir)
                            shutdown_start_time = time.time()
                        else:
                            time_since_last_learner_done_check = time.time()
                elif (time.time() - shutdown_start_time) > states.DURATION_SHUTDOWN_DELAY_TF:

                    # drain the event queues
                    if os.path.exists(self.log_dir):
                        etb_logger.info("final drain of trackers")
                        count_shut_down_reports = 0
                        # Drain the tracker queue
                        while self.track_and_report_one(report_only_if_queue_full=False):
                            # loop until track_and_report_one says it didn't report anything
                            count_shut_down_reports += 1
                        etb_logger.info("Drain flush: %d", count_shut_down_reports)
                    else:
                        etb_logger.info("No log_dir, abandoning trackers")
                        self.td_client.set_em_file_path(str(None))

                    etb_logger.info("Flushing buffer with force")
                    self.td_client.flush_maybe(force=True)

                    etb_logger.info("Having a little snooze for %f seconds", states.SLEEP_BEFORE_LC_DONE)
                    time.sleep(states.SLEEP_BEFORE_LC_DONE)

                    break  # From main loop

                self.td_client.flush_maybe()

                # thread yield
                time.sleep(0)
        finally:
            etb_logger.info("signaling that log-collector is done")
            states.signal_lc_done(logger=etb_logger)
            # I seem to have problems exiting directly, so, this sleep seems to help.
            # My unsubstantiated theory is that gRPC needs time to flush.
            # Note since we signaled, we won't actually wait n seconds, the
            # job monitor will delete us.
            time.sleep(states.SLEEP_BEFORE_EXIT_TIME)


def isInList(label_event_list, label, step, wall_time):
    for eventSet in label_event_list:
        if eventSet.label == label and \
                eventSet.event.step == step and \
                eventSet.event.wall_time == wall_time:
            return True

    return False


class EventSet:

    def __init__(self, label: str, event: event_accumulator.ScalarEvent):
        self.label = label
        self.event = event
        self.wall_time = -1.0

    # def emitStatus(tracker, message):
    #     dt = datetime.datetime.now()
    #     rfcRFC3339Formatted = dt.isoformat("T") + "Z"
    #
    #     labelValueList = []
    #     labelValueList.append(["Message", message])
    #
    #     emitEvalMetric(tracker.log_dir, "Status", 0, rfcRFC3339Formatted,
    #                    labelValueList)


# Emit evaluation metrics for visibility to DLaaS clients
def emitEvalMetric(td_client: tdb.TrainingDataClientBuffered,
                   group_label: str,  # group label, likely test or train
                   iterStep,
                   timestamp,
                   values_dict,
                   rindex: int,
                   event_wall_time: float,
                   sub_identifier
                   ):
    """Push the processed metrics data to the metrics service"""
    try:
        etimes = dict()

        etimes['iteration'] = tdp.Any(type=tdp.Any.INT, value=str(iterStep))
        etimes['timestamp'] = tdp.Any(type=tdp.Any.STRING, value=timestamp)
        etimes['wall_time'] = tdp.Any(type=tdp.Any.FLOAT, value=str(event_wall_time))

        # # d = datetime.datetime.utcnow() # <-- get time in UTC
        # #     print d.isoformat("T") + "Z"
        # #     if timestamp == None:
        # #         timestamp = start_time + datetime.timedelta(seconds=rowdict['Seconds'])
        #
        # dict_MetricData['timestamp'] = timestamp
        # time=int(event_wall_time * 1000),

        etb_logger.debug("Creating emetrics record")
        emetrics = tdp.EMetrics(
            meta=tdp.MetaInfo(
                training_id=os.environ["TRAINING_ID"],
                time=extract_datetime.get_meta_timestamp(),
                rindex=int(rindex),
                subid=sub_identifier
            ),
            grouplabel=group_label,
            etimes=etimes,
            values=values_dict
        )

        if td_client is not None:
            etb_logger.debug("Calling AddEMetrics")
            td_client.AddEMetrics(emetrics)

    except Exception as inst:
        etb_logger.error("Unexpected error when attempting to send emetrics: %s", sys.exc_info()[0])
        print("Unexpected error when attempting to send emetrics:", sys.exc_info()[0])
        print(type(inst))
        print(inst.args)
        traceback.print_exc()
        print(inst)

        sys.stdout.flush()

    return rindex+1


def subdir_below_logdir(summary_dir: str, log_dir)->str:
    rel_to_logdir = str(summary_dir[len(log_dir)+1:])
    return rel_to_logdir.split(os.path.sep)[0]


def is_sub_run_dir(summary_dir: str, log_dir)->bool:
    subdir = subdir_below_logdir(summary_dir, log_dir)

    return subdir.isdigit() or (subdir.startswith("learner-") and subdir[len("learner-"):].isdigit())


def dir_below_logdir(summary_dir: str, log_dir)->str:
    subdir = subdir_below_logdir(summary_dir, log_dir)
    return os.path.normpath(os.path.join(log_dir, subdir))


def extract(log_dir: str, should_connect: bool=True):
    global etb_logger

    log_scanner = scan_log_dirs.LogScanner(should_connect=should_connect, logger=etb_logger)

    log_dir = os.path.normpath(log_dir)

    etb_logger.info("looping over %s (log_dir)", log_dir)

    while True:
        # The design allows for there to be top-level log files, as well as subdirectory logs that
        # contain some sub-run logs.  These are indexed in the training data service with the subdir
        # in the MetaInfo inner struct.  Because these directories can be built as we're processing,
        # we check all the paths on each iteration.
        # etb_logger.debug("considering if %s (log_dir) exists (top_level=%r)", log_dir, top_level)

        if not os.path.exists(log_dir):
            time.sleep(1)
            continue

        # etb_logger.debug("log_dir DOES exist: %s", log_dir)

        # logging.debug("calling scan for the log_scanner")
        log_scanner.scan(log_dir=log_dir,
                         is_log=match_log_file.is_log_file,
                         push_function=push_log_line.push,
                         should_loop=False)

        for summary_dir in event_multiplexer.GetLogdirSubdirectories(log_dir):
            summary_dir = os.path.normpath(summary_dir)

            run_id = ""
            top_log_dir = log_dir

            if is_sub_run_dir(summary_dir, log_dir):
                run_id = subdir_below_logdir(summary_dir, log_dir)
                top_log_dir = dir_below_logdir(summary_dir, log_dir)

            if top_log_dir not in log_scanner.log_runners:
                td_client = log_scanner.td_client
                if run_id != "":
                    # In this case, we'll establish a new buffered tds client, since it
                    # has to know which directory to write to.
                    etb_logger.info("Creating buffered client for run_id: %s", run_id)
                    td_client = tdb.TrainingDataClientBuffered(td_client.td_client)
                run_tracker = Run_tracker(
                    td_client=td_client,
                    log_dir=top_log_dir,
                    sub_identifier=run_id,
                    lines_counter=log_scanner.lines_counter)

                if run_tracker is not None:
                    log_scanner.log_runners[top_log_dir] = run_tracker
                    run_tracker.build_event_trackers()
                    run_tracker.start()

        time.sleep(.5)


# class NoEvilDirectoryWatcherMessages(logging.Filter):
#     def filter(self, record):
#         print("record: %r" % record)
#         sys.stdout.flush()
#
#         if 'No path found after' in record.getMessage():
#             return 0
#         else:
#             return 1

def main():
    global etb_logger

    # Have to keep this is WARN because the tensorboard code dumps mountains of stuff with INFO
    logging.basicConfig(format='%(filename)s %(funcName)s %(lineno)d: %(message)s', level=logging.WARN)

    etb_logger = logging.getLogger("extract_tb")
    etb_logger.setLevel(logging.INFO)
    # logging.getLogger().addFilter(NoEvilDirectoryWatcherMessages())

    parser = argparse.ArgumentParser()

    log_directory = os.environ["LOG_DIR"]
    # em_file = log_directory + "/evaluation-metrics.txt"

    etb_logger.info("LOG_DIR = %s", log_directory)

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='DLaaS log directory')

    # parser.add_argument('--em_file', type=str, default=em_file,
    #                     help='Evaluation metrics file')

    FLAGS, unparsed = parser.parse_known_args()

    etb_logger.info("FLAGS.log_dir = %s", FLAGS.log_dir)

    extract(FLAGS.log_dir)


if __name__ == '__main__':
    main()
