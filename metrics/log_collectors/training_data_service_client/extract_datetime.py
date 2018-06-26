#!/usr/bin/env python
"""Extract datetime from text line"""

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
import re
import datetime
import time
import threading

from dateutil.parser import parse as datetime_parser

regExTimeGLog_str = "^(?P<severity>[EFIW])" \
                    "(?P<month>\d\d)" \
                    "(?P<day>\d\d)\s" \
                    "(?P<hour>\d\d):(" \
                    "?P<minute>\d\d):" \
                    "(?P<second>\d\d)\." \
                    "(?P<microsecond>\d{6})\s+" \
                    "(?P<process_id>-?\d+)\s" \
                    "(?P<filename>[a-zA-Z<_][\w._<>-]+):" \
                    "(?P<line>\d+)\]\s"

regExTimeGLog = re.compile(regExTimeGLog_str)

regex_timestamp_iso8601_str = r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[.]\d+?Z|[-+]\d{2}:\d{2}?"

regex_timestamp_iso8601 = re.compile(regex_timestamp_iso8601_str)

last_time_lock = threading.Lock()  # type: threading.Lock


def now_in_milliseconds()->int:
    return int(time.time() * 1000)


last_time = now_in_milliseconds()


def get_meta_timestamp()->int:
    global last_time
    # We must ensure that timestamps are always unique.  We can improve upon this
    # a little by making a timestamp object, and then having loglines and emetrics
    # trap separately.  But, this should be ok for now.
    with last_time_lock:
        this_time = now_in_milliseconds()
        while last_time == this_time:
            this_time = now_in_milliseconds()
        last_time = this_time
        timestamp = this_time

    return timestamp


def get_log_created_year(input_file) -> int:
    """Get year from log file system timestamp
    """

    log_created_time = os.path.getctime(input_file)
    log_created_year = datetime.datetime.fromtimestamp(log_created_time).year
    return log_created_year


def to_timestamp(dt, epoch=datetime.datetime(1970, 1, 1)):
    td = dt - epoch
    # return td.total_seconds()
    return int((td.microseconds + (td.seconds + td.days * 86400) * 10**6) / 10**6)


def extract_datetime(line: str, year: int, existing_time: datetime = None, fuzzy: bool = True) -> (datetime, bool):
    """Try to get a datetime object from the long line

    Return some datetime, and a boolean value telling if the value was actually extracted from the line"""

    matches = regExTimeGLog.match(line)
    did_get_good_time = False
    dt = None
    # TODO: allow type that specifies what kind of parse should be attempted
    if matches is not None:
        dtd = matches.groupdict()
        # print(year)
        # print(int(dtd["month"]))
        # print(int(dtd["day"]))
        # print(int(dtd["hour"]))
        # print(int(dtd["minute"]))
        # print(int(dtd["second"]))
        # print(int(dtd["microsecond"]))
        dt = datetime.datetime(year,
                               int(dtd["month"]),
                               int(dtd["day"]),
                               int(dtd["hour"]),
                               int(dtd["minute"]),
                               int(dtd["second"]),
                               int(dtd["microsecond"]))
    else:
        try:
            # I'd like to be less restrictive than iso8601, but, this is what we do for now.
            matches = regex_timestamp_iso8601.match(line)
            if matches is not None:
                dt = datetime_parser(line, fuzzy=fuzzy)
        except ValueError:
            dt = None

    if dt is not None:
        did_get_good_time = True
    elif existing_time is None:
        dt = datetime.datetime.now()
    else:
        dt = existing_time

    return dt, did_get_good_time
