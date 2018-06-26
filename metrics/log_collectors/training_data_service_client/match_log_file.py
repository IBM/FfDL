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
import logging

LOGFILE_DEFAULT_NAME = "training-log.txt"
LOGFILE_ALT1_NAME = "user_log.txt"

EMETRICS_FILE_BASE_NAME = "evaluation-metrics.txt"

# This is experimental, and for now I'm hard coding it.  Once settled,
# we'll allow this array to be set by the manifest.
possible_log_file_names = [LOGFILE_DEFAULT_NAME, LOGFILE_ALT1_NAME]


def is_log_file(filename):
    for log_file_pattern in possible_log_file_names:
        if filename == log_file_pattern:
            return True
    return False


def is_emetrics_file(filename):
    return EMETRICS_FILE_BASE_NAME == filename
