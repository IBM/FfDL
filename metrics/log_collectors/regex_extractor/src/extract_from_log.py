#!/usr/bin/env python
"""Default incremental log parser for Caffe deep learning jobs."""

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

import yaml
import os
import re
import time
import datetime
import argparse
import sys

from typing import List, Dict

from log_collectors.training_data_service_client import print_json as print_json

from log_collectors.training_data_service_client import training_data_pb2 as tdp
from log_collectors.training_data_service_client import connect as connect

from log_collectors.training_data_service_client import extract_datetime as extract_datetime

from log_collectors.training_data_service_client import push_log_line as push_log_line


def read_symbol_libs(dir_path: str) -> Dict[str, str]:
    patterns_dir = os.path.join(dir_path, "patterns")

    symbol_dict: dict = dict()
    pattern_lib_regex = r"(?P<name>\w+)\s(?P<pattern>.*)"
    for subdir, dirs, files in os.walk(patterns_dir):
        for file in files:
            pattern_file_path: str = os.path.join(subdir, file)
            with open(pattern_file_path, 'rt') as f:
                for line in iter(f):
                    line = line.strip()
                    if line.startswith("#"):
                        continue
                    if line == "":
                        continue
                    match = re.match(pattern_lib_regex, line)
                    if match is not None:
                        match_dict = match.groupdict()
                        symbol_dict[match_dict.get("name")] = match_dict.get("pattern")
                        # print("found: "+match_dict.get("name")+": "+patterns_dict.get(match_dict.get("name")))
    return symbol_dict


def replace_lib_symbols(symbol_dict: dict, regex_val: str) -> str:
    """Recursively substitute library symbols

    Return string that should have all it's symbols replaced."""

    pattern_lib_ref_regex = r"%{(?P<type>\w+)}"
    pattern_lib_ref_specific_form = "(?P<type>%s)"
    subst_form = "%s"

    rebuilt_regex = regex_val
    matches = re.finditer(pattern_lib_ref_regex, rebuilt_regex)

    for _, match in enumerate(matches):
        match_pairs = match.groupdict()
        symbol_name = match_pairs["type"]

        lib_regex = symbol_dict[symbol_name]
        pattern_lib_ref_specific = pattern_lib_ref_specific_form % symbol_name
        pattern_lib_ref_specific = "%{" + pattern_lib_ref_specific + "}"
        subst = subst_form % lib_regex
        rebuilt_regex = re.sub(pattern_lib_ref_specific, subst, rebuilt_regex, 1)

        rebuilt_regex = replace_lib_symbols(symbol_dict, rebuilt_regex)

    return rebuilt_regex


def read_extract_description(manifest: str, symbol_dict: dict) -> dict:
    pattern_lib_ref_regex = r"%{(?P<type>\w+):(?P<match_name>\w+)}"
    pattern_lib_ref_specific_form = "(?P<type>%s):(?P<match_name>\w+)"
    subst_form = "(?P<%s>%s)"

    evaluation_metrics: dict = None
    with open(manifest, 'r') as stream:
        try:
            manifest_data: dict = yaml.load(stream)
            evaluation_metrics = manifest_data["evaluation_metrics"]

            groups: dict = evaluation_metrics["groups"]
            for group_key in groups:
                group = groups[group_key]
                regex_val: str = group.get("regex")
                rebuilt_regex = regex_val
                matches = re.finditer(pattern_lib_ref_regex, rebuilt_regex)
                for _, match in enumerate(matches):
                    match_pairs = match.groupdict()
                    symbol_name = match_pairs["type"]
                    match_name = match_pairs["match_name"]

                    lib_regex = symbol_dict[symbol_name]
                    pattern_lib_ref_specific = pattern_lib_ref_specific_form % match_pairs["type"]
                    pattern_lib_ref_specific = "%{" + pattern_lib_ref_specific + "}"
                    subst = subst_form % (match_name, lib_regex)
                    rebuilt_regex = re.sub(pattern_lib_ref_specific, subst, rebuilt_regex, 1)

                rebuilt_regex = replace_lib_symbols(symbol_dict, rebuilt_regex)
                group["regex_expanded"] = re.compile(rebuilt_regex, re.MULTILINE | re.DOTALL)

        except yaml.YAMLError as exc:
            print(exc)

    return evaluation_metrics


def type_string_to_grpc_type(value_type: str) -> str:
    grpc_value_type = ""
    if value_type == "INT":
        grpc_value_type = tdp.Any.INT
    elif value_type == "FLOAT":
        grpc_value_type = tdp.Any.FLOAT
    elif value_type == "STRING":
        grpc_value_type = tdp.Any.STRING
    elif value_type == "JSONSTRING":
        grpc_value_type = tdp.Any.INT

    return grpc_value_type

def extract(em_file_path: str, manifest: str, follow: bool, should_connect: bool=True):
    dir_path = os.path.dirname(os.path.realpath(__file__))
    symbol_dict: Dict[str, str] = read_symbol_libs(dir_path)

    evaluation_metrics_spec = read_extract_description(manifest, symbol_dict)

    logfile = evaluation_metrics_spec["in"]

    job_directory = os.environ["JOB_STATE_DIR"]
    regex = r"\$JOB_STATE_DIR"
    logfile = re.sub(regex, job_directory, logfile, 0)

    # Not sure why I seem to loose the under-bar somewhere along the line.
    if "line_lookahead" in evaluation_metrics_spec:
        line_lookahead: int = int(evaluation_metrics_spec["line_lookahead"])
    elif "linelookahead" in evaluation_metrics_spec:
        line_lookahead: int = int(evaluation_metrics_spec["linelookahead"])
    else:
        line_lookahead: int = 4

    groups: dict = evaluation_metrics_spec["groups"]

    line_length_stack: List[int] = []
    text_window = ""
    record_index = 0
    read_pos = 0
    line_index = 1

    learner_job_is_running = True
    logfile_year = None
    start_time: datetime = None
    did_get_good_time: bool = False

    if should_connect:
        tdClient = connect.get_connection()
    else:
        tdClient = None

    while learner_job_is_running:
        if os.path.exists(logfile):
            if logfile_year is None:
                logfile_year = extract_datetime.get_log_created_year(logfile)

            with open(logfile, 'r') as log_stream:
                log_stream.seek(read_pos)

                try:
                    for line in iter(log_stream):
                        # Do our best to get a good start time.
                        if not did_get_good_time:
                            # keep trying to get a good start time from the log line, until it's pointless
                            start_time, did_get_good_time = \
                                extract_datetime.extract_datetime(line, logfile_year, start_time)

                        line_index = push_log_line.push(tdClient, line, logfile_year, line_index)

                        line_length_stack.append(len(line))
                        text_window += line
                        if len(line_length_stack) > line_lookahead:
                            length_first_line = line_length_stack[0]
                            line_length_stack = line_length_stack[1:]
                            text_window = text_window[length_first_line:]

                        for group_key in groups:
                            group = groups[group_key]
                            name = group_key
                            regex_expanded = group["regex_expanded"]
                            matches = regex_expanded.match(text_window)
                            if matches is not None:
                                values_dict = matches.groupdict()

                                # meta_dict_desc = group["meta"]
                                etimes_descriptions: dict = group["etimes"]
                                if etimes_descriptions is None:
                                    print("Did not find etimes! Found: ")
                                    for axis_key in group:
                                        print("key: "+axis_key)
                                        sys.stdout.flush()
                                    break

                                etimes: dict = dict()
                                for etime_key in etimes_descriptions:
                                    item = etimes_descriptions[etime_key]
                                    valOrRef: str = item["value"]
                                    if valOrRef.startswith("$"):
                                        value_inner = valOrRef[1:]
                                        value_actual = values_dict[value_inner]
                                    else:
                                        value_actual = valOrRef
                                    grpc_value_type = type_string_to_grpc_type(item["type"])
                                    etimes[etime_key] = tdp.Any(type=grpc_value_type, value=value_actual)

                                if "scalars" in group:
                                    scalars_descriptions: dict = group["scalars"]
                                elif "values" in group:
                                    scalars_descriptions: dict = group["values"]
                                else:
                                    scalars_descriptions = None

                                if scalars_descriptions is None:
                                    print("Did not find scalars! Found: ")
                                    for axis_key in group:
                                        print("key: "+axis_key)
                                        sys.stdout.flush()
                                    break

                                scalars: dict = dict()
                                for scalar_key in scalars_descriptions:
                                    item = scalars_descriptions[scalar_key]
                                    valOrRef: str = item["value"]
                                    if valOrRef.startswith("$"):
                                        value_inner = valOrRef[1:]
                                        value_actual = values_dict[value_inner]
                                    else:
                                        value_actual = valOrRef
                                    value_type = item["type"]
                                    grpc_value_type = type_string_to_grpc_type(value_type)
                                    scalars[scalar_key] = tdp.Any(type=grpc_value_type, value=value_actual)

                                date_string: str = line
                                if "meta" in group:
                                    meta_list: dict = group["meta"]
                                    if "time" in meta_list:
                                        valOrRef: str = meta_list["time"]
                                        if valOrRef.startswith("$"):
                                            value_ref = valOrRef[1:]
                                            date_string = values_dict[value_ref]
                                        else:
                                            date_string = valOrRef

                                # At this point, don't keep trying to get a start time if we haven't already
                                did_get_good_time = True
                                # TODO: pass in the type specified by the regex
                                line_time, _ = extract_datetime.extract_datetime(date_string, logfile_year, None)
                                microseconds = (line_time - start_time).microseconds
                                timestamp = int(microseconds)
                                record_index += 1
                                emetrics = tdp.EMetrics(
                                    meta=tdp.MetaInfo(
                                        training_id=os.environ["TRAINING_ID"],
                                        time=timestamp,
                                        rindex=record_index
                                    ),
                                    grouplabel=name,
                                    etimes=etimes,
                                    values=scalars
                                )

                                json_form = print_json.to_string(emetrics)

                                with open(em_file_path, 'a') as em_stream:
                                    em_stream.write(json_form)
                                    em_stream.write("\n")

                                if tdClient is not None:
                                    tdClient.AddEMetrics(emetrics)

                                # for now, print to stdout (support old endpoint).
                                # TODO: Don't print to stdout for metrics
                                print(json_form)

                                text_window = ""
                                line_length_stack = []
                                break

                except Exception as inst:
                    print("Unexpected error when attempting to process evaluation metric record:",
                          sys.exc_info()[0])
                    print(inst)
                    sys.stdout.flush()

                read_pos = log_stream.tell()

            learner_job_is_running = follow

        # wait a second before reading the file again
        # (unless you want to constantly check the logs for new content?)
        time.sleep(1)


def main():
    job_directory = os.environ["JOB_STATE_DIR"]
    log_directory = job_directory + "/logs"
    em_file = log_directory + "/evaluation-metrics.txt"

    manifest = os.environ["EM_DESCRIPTION"]

    parser = argparse.ArgumentParser()

    parser.add_argument('--manifest', type=str, default=manifest,
                        help='DLaaS log directory')

    parser.add_argument('--em_file', type=str, default=em_file,
                        help='Evaluation metrics file')

    FLAGS, _ = parser.parse_known_args()

    extract(FLAGS.em_file, FLAGS.manifest, True, True)


if __name__ == '__main__':
    main()
