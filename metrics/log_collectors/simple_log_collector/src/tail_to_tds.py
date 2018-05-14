#!/usr/bin/env python
import os
import argparse
import logging

from log_collectors.training_data_service_client import match_log_file
from log_collectors.training_data_service_client import push_log_line
from log_collectors.training_data_service_client import scan_log_dirs


def main():
    logging.basicConfig(format='%(filename)s %(funcName)s %(lineno)d: %(message)s', level=logging.INFO)
    log_directory = os.environ["LOG_DIR"]
    # log_file = log_directory + "/latest-log"

    parser = argparse.ArgumentParser()

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='Log directory')

    FLAGS, unparsed = parser.parse_known_args()

    scan_log_dirs.LogScanner(should_connect=True).scan(
        log_dir=FLAGS.log_dir,
        is_log=match_log_file.is_log_file,
        push_function=push_log_line.push)


if __name__ == '__main__':
    main()
