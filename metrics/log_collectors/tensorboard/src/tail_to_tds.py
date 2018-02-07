#!/usr/bin/env python
import os
import argparse

from log_collectors.training_data_service_client import tail_to_tds as tail


def main():
    job_directory = os.environ["JOB_STATE_DIR"]
    log_directory = job_directory + "/logs"
    log_file = job_directory + "/latest-log"

    parser = argparse.ArgumentParser()

    parser.add_argument('--log_file', type=str, default=log_file,
                        help='Log file')

    FLAGS, _ = parser.parse_known_args()

    tail.collect_and_send(FLAGS.log_file, True)


if __name__ == '__main__':
    main()
