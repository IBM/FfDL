#!/usr/bin/env python
"""Print json to stdout or wherever"""

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

import re
import logging
from google.protobuf import json_format

from builtins import object

rindex_regex = re.compile(r"\"rindex\":\s\"([0-9]+)\"")
time_regex = re.compile(r"\"time\":\s\"([0-9]+)\"")


def output(obj: object):
    """Take a EMetrics or LogLine record and print a valid JSON object to stdout"""
    json_string = to_string(obj)
    print(json_string)


def logging_output(obj: object):
    """Take a EMetrics or LogLine record and print a valid JSON object to stdout"""
    json_string = to_string(obj)
    logging.debug("json: %s", json_string)


def to_string(obj: object)->str:
    """Take a EMetrics or LogLine record and return a valid JSON string"""

    # Call buggy google protobuf function.
    json_string = json_format.MessageToJson(obj, indent=0, preserving_proto_field_name=True)

    # The rest of this is a bit of a hack.  Perhaps I'd be better off just
    # processing the typed record and hand-printing the json.
    json_string = json_string.replace('\n', ' ').replace('\r', '')

    for i in range(1, 10):
        json_string = json_string.replace('  ', ' ')
        json_string = json_string.replace('{ "', '{"')
        json_string = json_string.replace('" }', '"}')

    json_string = json_string.replace('"type": "STRING"', '"type": 0')
    json_string = json_string.replace('"type": "JSONSTRING"', '"type": 1')
    json_string = json_string.replace('"type": "INT"', '"type": 2')
    json_string = json_string.replace('"type": "FLOAT"', '"type": 3')

    json_string = rindex_regex.sub(r'"rindex": \1', json_string)
    json_string = time_regex.sub(r'"time": \1', json_string)

    return json_string
