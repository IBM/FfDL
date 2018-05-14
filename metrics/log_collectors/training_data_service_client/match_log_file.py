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
