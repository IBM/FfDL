#!/usr/bin/env bash

eval_metrics="$JOB_STATE_DIR/logs/evaluation-metrics.txt"
summary_metrics_file="$JOB_STATE_DIR/logs/summary-metrics.txt"

export PYTHONPATH=$PYTHONPATH:$PWD

echo Evaluation metrics using tensorflow version: $TENSORFLOWVERSION, python version: $PYTHONVERSION

while [ ! -f $JOB_STATE_DIR/latest-log ]
do
  sleep 1
done

python tail_to_tds.py &
python extract_tb.py
