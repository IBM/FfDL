#!/usr/bin/env python
import os
import argparse

from log_collectors.training_data_service_client import tail_to_em_from_emfile


def main():
    job_directory = os.environ["JOB_STATE_DIR"]
    log_directory = job_directory + "/logs"
    em_file = log_directory + "/evaluation-metrics.txt"

    parser = argparse.ArgumentParser()

    parser.add_argument('--em_file', type=str, default=em_file,
                        help='Evaluation metrics file')

    parser.add_argument('--should_connect', type=bool, default=True,
                        help='If true send data to gRPC endpoint')

    parser.add_argument('--send', dest='send', action='store_true')
    parser.add_argument('--no-send', dest='send', action='store_false')
    parser.set_defaults(send=True)

    FLAGS, _ = parser.parse_known_args()

    print("Should connect: "+str(FLAGS.should_connect))

    tail_to_em_from_emfile.collect_and_send(FLAGS.em_file, FLAGS.send)


if __name__ == '__main__':
    main()
