#!/usr/bin/env python
import os
import argparse
import logging

from log_collectors.training_data_service_client import scan_log_dirs
from log_collectors.training_data_service_client import match_log_file
from log_collectors.training_data_service_client import push_log_line
from log_collectors.training_data_service_client import push_em_line


def main():
    logging.basicConfig(format='%(filename)s %(funcName)s %(lineno)d: %(message)s', level=logging.INFO)

    log_directory = os.environ["LOG_DIR"]
    # log_file = log_directory + "/latest-log"

    parser = argparse.ArgumentParser()

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='Log directory')

    parser.add_argument('--should_connect', type=bool, default=True,
                        help='If true send data to gRPC endpoint')

    parser.add_argument('--send', dest='send', action='store_true')
    parser.add_argument('--no-send', dest='send', action='store_false')
    parser.set_defaults(send=True)

    FLAGS, _ = parser.parse_known_args()

    logging.info("Should connect: "+str(FLAGS.should_connect))

    scan_log_dirs.LogScanner(should_connect=True).scan(
        log_dir=FLAGS.log_dir,
        is_log=match_log_file.is_log_file,
        push_function=push_log_line.push,
        is_emetrics=match_log_file.is_emetrics_file,
        push_emetrics_function=push_em_line.push)


if __name__ == '__main__':
    main()
