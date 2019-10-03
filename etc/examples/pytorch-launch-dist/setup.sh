if [ "$LEARNER_ID" == "1" ]; then
  echo $(hostname --ip-address) > /job/master_node.txt
  export master_node=$(hostname --ip-address)
else
  while [ -z $master_node ]; do
    export master_node=$(cat /job/master_node.txt)
    sleep 5
  done
fi;
export node_rank="$(($LEARNER_ID-1))"
export NUM_GPUS=${GPU_COUNT%.*}
printenv | sed 's/^/export /;s/=/=\"/;s/$/\"/' > env_file
