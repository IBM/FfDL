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
import datetime
import argparse
import sys
import logging

from typing import List, Dict, Any, Tuple

from log_collectors.training_data_service_client import training_data_pb2 as tdp
from log_collectors.training_data_service_client import training_data_buffered as tdb

from log_collectors.training_data_service_client import extract_datetime as edt

from log_collectors.training_data_service_client import push_log_line as push_log_line
from log_collectors.training_data_service_client import match_log_file
from log_collectors.training_data_service_client import scan_log_dirs


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


class ExtraPushData4RegExExtractor:
    def __init__(self, line_lookahead: int, groups: dict):
        self.line_lookahead = line_lookahead
        self.groups = groups

        self.line_length_stack: List[int] = []
        self.text_window = ""
        self.start_time: datetime = None
        self.did_get_good_time: bool = False


def extract_and_push_emetrics(td_client: tdb.TrainingDataClientBuffered, log_file: str,
                              log_line: str, logfile_year: str,
                              line_index: int, record_index: int,
                              subdir: str, extra_any: Any=None) -> Tuple[int, int]:

    state: ExtraPushData4RegExExtractor = extra_any
    # # Do our best to get a good start time.
    # if not state.did_get_good_time:
    #     # keep trying to get a good start time from the log line, until it's pointless
    #     start_time, did_get_good_time = \
    #         edt.extract_datetime(log_line, logfile_year, state.start_time)

    if td_client.em_file_path is None:
        em_file_path = os.path.join(os.path.dirname(log_file), match_log_file.EMETRICS_FILE_BASE_NAME)
        td_client.set_em_file_path(em_file_path)
        logging.debug("em_file_path: %s, subdir: %s", td_client.em_file_path, subdir)

    # logging.debug("push_log_line.push %d (subdir: %s) %s", line_index, subdir, log_line)
    line_index, _ = push_log_line.push(td_client, log_file, log_line, logfile_year,
                                       line_index, 0, subdir, state)

    state.line_length_stack.append(len(log_line))
    state.text_window += log_line
    if len(state.line_length_stack) > state.line_lookahead:
        length_first_line = state.line_length_stack[0]
        state.line_length_stack = state.line_length_stack[1:]
        state.text_window = state.text_window[length_first_line:]

    logging.debug("len groups: %d", len(state.groups))
    for group_key in state.groups:
        group = state.groups[group_key]
        name = group_key
        regex_expanded = group["regex_expanded"]
        matches = regex_expanded.match(state.text_window)
        if matches is not None:
            values_dict = matches.groupdict()

            # meta_dict_desc = group["meta"]
            etimes_descriptions: dict = group["etimes"]
            if etimes_descriptions is None:
                logging.warning("Did not find etimes! Found: ")
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
                logging.warning("Did not find scalars! Found: ")
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

            # date_string: str = line
            subid_value = subdir
            if "meta" in group:
                meta_list: dict = group["meta"]
                # if "time" in meta_list:
                #     valOrRef: str = meta_list["time"]
                #     if valOrRef.startswith("$"):
                #         value_ref = valOrRef[1:]
                #         # date_string = values_dict[value_ref]
                #     else:
                #         # date_string = valOrRef
                #         pass
                if "subid" in meta_list:
                    valOrRef: str = meta_list["subid"]
                    if valOrRef.startswith("$"):
                        logging.debug("resetting subdir(subid): %s, metalist: %r",
                                      subid_value, meta_list)
                        value_ref = valOrRef[1:]
                        subid_value = values_dict[value_ref]
                    elif not valOrRef == "":
                        logging.debug("resetting subdir(subid): %s, metalist: %r",
                                      subid_value, meta_list)
                        subid_value = valOrRef

            logging.debug("about to push evaluation metrics with subdir(subid): %s", subid_value)

            # At this point, don't keep trying to get a start time if we haven't already
            state.did_get_good_time = True
            emetrics = tdp.EMetrics(
                meta=tdp.MetaInfo(
                    training_id=os.environ["TRAINING_ID"],
                    time=edt.get_meta_timestamp(),
                    rindex=record_index,
                    subid=subid_value
                ),
                grouplabel=name,
                etimes=etimes,
                values=scalars
            )
            record_index += 1
            # state.total_lines_pushed += 1

            if td_client is not None:
                td_client.AddEMetrics(emetrics)

            # for now, print to stdout (support old endpoint).
            # print(json_form)

            state.text_window = ""
            state.line_length_stack = []
            break

    return line_index, record_index


def extract(log_dir: str, manifest: str, should_connect: bool=True):
    dir_path = os.path.dirname(os.path.realpath(__file__))
    symbol_dict: Dict[str, str] = read_symbol_libs(dir_path)

    evaluation_metrics_spec = read_extract_description(manifest, symbol_dict)

    # Not sure why I seem to loose the under-bar somewhere along the line.
    if "line_lookahead" in evaluation_metrics_spec:
        line_lookahead: int = int(evaluation_metrics_spec["line_lookahead"])
    elif "linelookahead" in evaluation_metrics_spec:
        line_lookahead: int = int(evaluation_metrics_spec["linelookahead"])
    else:
        line_lookahead: int = 4

    groups: dict = evaluation_metrics_spec["groups"]

    logging.debug("log dir: %s" % log_dir)

    extraData = ExtraPushData4RegExExtractor(line_lookahead, groups)

    scan_log_dirs.LogScanner(should_connect=should_connect).scan(
        log_dir=log_dir,
        extra=extraData,
        is_log=match_log_file.is_log_file,
        push_function=extract_and_push_emetrics)


def main():
    logging.basicConfig(format='%(filename)s %(funcName)s %(lineno)d: %(message)s', level=logging.INFO)
    log_directory = os.environ["LOG_DIR"]
    # log_file = log_directory + "/latest-log"
    # em_file = log_directory + "/evaluation-metrics.txt"

    manifest = os.environ["EM_DESCRIPTION"]

    parser = argparse.ArgumentParser()

    parser.add_argument('--manifest', type=str, default=manifest,
                        help='DLaaS log directory')

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='Log directory')

    FLAGS, _ = parser.parse_known_args()

    extract(FLAGS.log_dir, FLAGS.manifest, True)

    logging.info("Normal exit")


if __name__ == '__main__':
    main()
