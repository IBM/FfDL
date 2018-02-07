#!/usr/bin/env python
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import sys
import os
import datetime
import traceback


from log_collectors.training_data_service_client import connect as connect
from log_collectors.training_data_service_client import training_data_pb2 as tdp

if os.environ.get("TENSORFLOWVERSION") == "1.2":
    from tensorflow.tensorboard.backend.event_processing import event_accumulator
else:
    from tensorboard.backend.event_processing import event_accumulator

from log_collectors.training_data_service_client import print_json


class Tracker:
    ea = None
    tb_log_dir = ""
    log_dir = ""

    # tdClient: training_data_service_client.training_data_pb2_grpc.TrainingDataStub
    tdClient = None

    # this is the last reported events, which we need to avoid reporting
    # multiple times.
    #     label_event_list = []

    # dict by step key, value is label_event_list
    # I don't think I need this to be a dict, and I probably just
    # have to keep the latest set to check.
    # But, keeping it as is for the moment.
    iterDict = dict()


def get_summary_types(tb_log_dir):
    tb_log_dir = tb_log_dir + '/'
    if os.path.isdir(tb_log_dir):
        return os.listdir(tb_log_dir)
    else:
        return []
    # for dir in os.listdir(tb_log_dir):
    #     print(dir)
    #     sys.stdout.flush()


# def trackEvents(tb_log_dir: str,
#                    log_dir: str,
#                    summary_type: str,
#                    tdClient: training_data_service_client.training_data_pb2_grpc.TrainingDataStub):
def trackEvents(
        tb_log_dir,
        log_dir,
        summary_type,
        tdClient):
    """Return tracker object (meant to be opaque) to be passed to emitEvalMetricViaTracker(...)"""
    summary_dir = tb_log_dir + '/' + summary_type + '/'
    ea = event_accumulator.EventAccumulator(summary_dir)
    # ea = event_accumulator.EventAccumulator(summary_dir,
    #     size_guidance={
    #                    event_accumulator.COMPRESSED_HISTOGRAMS: 1,
    #                    event_accumulator.IMAGES: 1,
    #                    event_accumulator.AUDIO: 1,
    #                    event_accumulator.SCALARS: 1000,
    #                    event_accumulator.HISTOGRAMS: 1,
    #                    })

    tracker = Tracker()
    tracker.ea = ea
    tracker.tb_log_dir = tb_log_dir
    tracker.log_dir = log_dir
    tracker.tdClient = tdClient
    tracker.group = summary_type

    return tracker


def isInList(label_event_list, label, step):
    for eventSet in label_event_list:
        if eventSet.label == label and eventSet.event.step == step:
            return True

    return False


class EventSet:
    label = ""
    event = None
    reported = False

# def emitEvalMetricMessage(log_dir, iterStep, message):
#     dict_MetricData = dict()
#     dict_MetricData['type'] = "message"
#
#     dict_MetricData['iteration'] = iterStep
#
#     dict_MetricValues = dict()
#
#     dict_MetricValues['message'] = message
#
#     dict_MetricData['values'] = dict_MetricValues
#
#     metricRecordStr = json.dumps(dict_MetricData, sort_keys=False,
#                              separators=(',\t', ': '))
#
#     path_to_eval_metrics = log_dir + '/' + 'evaluation-metrics.txt'
#     #     print("writing to "+path_to_eval_metrics)
#     with open(path_to_eval_metrics, "a+") as f:
#         f.write(metricRecordStr)
#         f.write("\n")
#         f.flush()
#
#     sys.stdout.write(metricRecordStr)
#     sys.stdout.write("\n")
#     sys.stdout.flush()


emitIntermediateDiagnostics = False

emitTagInfo = False


def diagTagInfo(str):
    if emitTagInfo:
        print(str)
        sys.stdout.flush()


# Emit evaluation metrics from a tracker object, for visibility to DLaaS clients.
def emitEvalMetricViaTracker(em_file_path: str, tracker, iterStep, reportInterval,
                             remainderCompare, rindex):

    ea = tracker.ea
    ea.Reload() # loads events from file

    # if emitIntermediateDiagnostics:
    #     emitEvalMetricMessage(tracker.log_dir, iterStep,
    #                           "skimming at iterStep #"+str(iterStep))
    #
    #     # rootScalerRecord = ea.Scalars(labelKeyList[0][1])[0]
    #     # print(rootScalerRecord)

    tags = ea.Tags()
    scaler_keys = tags[event_accumulator.SCALARS]
    for scaler_key in scaler_keys:

        scalerEvents = ea.Scalars(scaler_key)

        # print(type(scalerEvents))

        for event in scalerEvents:
            # time.sleep(0.1)
            label_event_list = tracker.iterDict.get(str(event.step))
            if label_event_list == None:
                eventSet = EventSet()
                eventSet.label = scaler_key
                eventSet.event = event
                tracker.iterDict[str(event.step)] = [eventSet]
            else:
                if isInList(label_event_list, scaler_key, event.step) == True:
                    #already been reported, continue
                    continue
                eventSet = EventSet()
                eventSet.label = scaler_key
                eventSet.event = event
                tracker.iterDict[str(event.step)].append(eventSet)

    latest_iter = iterStep

    for key in tracker.iterDict.keys():
        labelValueList = {}
        eventList = tracker.iterDict[key]
        foundReportable = False
        rfcRFC3339Formatted = None
        eventWallTime = None

        for eventSet in eventList:
            if eventSet.reported == False:
                foundReportable = True
                label = eventSet.label;
                event = eventSet.event
                iterStep = event.step
                eventWallTime = event.wall_time
                if iterStep > latest_iter:
                    latest_iter = iterStep
                if rfcRFC3339Formatted == None:
                    dt = datetime.datetime.fromtimestamp(event.wall_time)
                    rfcRFC3339Formatted = dt.isoformat("T") + "Z"

                labelValueList[label] = tdp.Any(type=tdp.Any.FLOAT, value=str(event.value))
                eventSet.reported = True

        if reportInterval > 0 and ((iterStep % reportInterval) != remainderCompare):
            foundReportable = False

        if foundReportable:
            emitEvalMetric(em_file_path, tracker.log_dir, tracker.tdClient, tracker.group, iterStep, rfcRFC3339Formatted,
                           labelValueList, rindex, eventWallTime)
            rindex+=1

        # time.sleep(0.1)

    return latest_iter, rindex

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
def emitEvalMetric(em_file_path: str, log_dir, td_client, group_label, iterStep, timestamp, values_dict, rindex, eventWallTime):
    '''Push the processed metrics data to the metrics service'''
    try:

        etimes = dict()

        etimes['iteration'] = tdp.Any(type=tdp.Any.INT, value=str(iterStep))
        etimes['timestamp'] = tdp.Any(type=tdp.Any.STRING, value=timestamp)

        # # d = datetime.datetime.utcnow() # <-- get time in UTC
        # #     print d.isoformat("T") + "Z"
        # #     if timestamp == None:
        # #         timestamp = start_time + datetime.timedelta(seconds=rowdict['Seconds'])
        #
        # dict_MetricData['timestamp'] = timestamp

        emetrics = tdp.EMetrics(
            meta=tdp.MetaInfo(
                training_id=os.environ["TRAINING_ID"],
                time=int(eventWallTime),
                rindex=int(rindex)
            ),
            grouplabel=group_label,
            etimes=etimes,
            values=values_dict
        )

        if td_client is not None:
            td_client.AddEMetrics(emetrics)

        json_form = print_json.to_string(emetrics)

        with open(em_file_path, 'a') as em_stream:
            em_stream.write(json_form)
            em_stream.write("\n")

        # for now, print to stdout.
        # TODO: Don't print to stdout for metrics
        print(json_form)

    except Exception as inst:
        print("Unexpected error when attempting to send emetrics:", sys.exc_info()[0])
        print(type(inst))
        print(inst.args)
        traceback.print_exc()
        print(inst)

        sys.stdout.flush()

    return rindex+1


def extract(em_file_path: str, tb_log_dir: str, log_dir: str, should_connect: bool=True):

    if should_connect:
        tdClient = connect.get_connection()
    else:
        tdClient = None

    groups = get_summary_types(tb_log_dir)

    event_trackers = []
    for group in groups:
        sys.stdout.flush()
        ea = trackEvents(tb_log_dir, log_dir, group, tdClient)
        event_trackers.append(ea)

    iterTest = [0] * len(event_trackers)

    record_index = 1

    while True:
        for idx,ea in enumerate(event_trackers):
            iterTest[idx], record_index = emitEvalMetricViaTracker(em_file_path, ea,  iterTest[idx], 1, 0, record_index)

        groups_new = get_summary_types(tb_log_dir)
        for idx, group in enumerate(groups_new):
            if group not in groups:
                groups.append(group)
                ea = trackEvents(tb_log_dir, log_dir, group, tdClient)
                event_trackers.append(ea)
                iterTest.append(0)


def main():
    parser = argparse.ArgumentParser()

    job_directory = os.environ["JOB_STATE_DIR"]
    log_directory = job_directory + "/logs"
    em_file = log_directory + "/evaluation-metrics.txt"

    log_directory = job_directory + "/logs"

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='DLaaS log directory')
    parser.add_argument('--tb_log_dir', type=str, default=log_directory+'/tb',
                        help='Summaries log directory')

    parser.add_argument('--em_file', type=str, default=em_file,
                        help='Evaluation metrics file')

    FLAGS, unparsed = parser.parse_known_args()

    extract(FLAGS.em_file, FLAGS.tb_log_dir, FLAGS.log_dir)


if __name__ == '__main__':
    main()
