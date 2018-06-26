#!/usr/bin/env python

import os
import subprocess
import sys

#wrapper loop which will run the actual training command
def run_training(cmd):
   process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
   print ("process should have started executing now... {}".format(" ".join(cmd)))
   while process.poll() is None:
      output = process.stdout.readline()
      if output:
         print(output.strip().decode("utf-8"))
         sys.stdout.flush()
   rc = process.returncode
   if rc != 0:
      raise ValueError("cmd: {0} exited with code {1}".format(" ".join(cmd), rc)) 
            

if __name__ =="__main__":
    number_of_learners_str = os.getenv("NUM_LEARNERS") 
    number_of_learners = int(number_of_learners_str)

    learner_name_prefix = os.getenv("LEARNER_NAME_PREFIX") #base name for all learners created, individual learners as separate by <>-1, <>-2 and so on
    number_of_ps_hosts_str = os.getenv("PS_HOSTS_COUNT",1)
    number_of_ps_hosts = int(number_of_ps_hosts_str)
    
    learner_id_str = os.getenv("LEARNER_ID") # id for the learner. depending on the learner on which this code executes you get a value of  1, 2, 3..
    learner_id = int(learner_id_str)

    # TODO 
    # error out if any of the above values are not defined

    job_name = "ps" if int(learner_id) <= number_of_ps_hosts else "worker"
    task_index = learner_id - 1  if job_name == "ps" else learner_id - 1 - number_of_ps_hosts  #learner_id starts from 1, so minus 1
    
    ps_hosts = []
    worker_hosts = []
    #host name needs to be in the format of host.domain learner_name_prefix-<id> is the host and learner_name_prefix is the domain
    # and hence the format learner_name_prefix-<id>.learner_name_prefix
    for i in range(0, number_of_ps_hosts):
        ps_hosts.append("{}-{}.{}:2222".format(learner_name_prefix, i, learner_name_prefix))

    for i in range(number_of_ps_hosts, number_of_learners):
       worker_hosts.append("{}-{}.{}:2222".format(learner_name_prefix, i, learner_name_prefix)) 
    
    command = sys.argv[1:] # base command
    command.append("--job_name={}".format(job_name))
    command.append("--ps_hosts={}".format(",".join(ps_hosts)))
    command.append("--worker_hosts={}".format(",".join(worker_hosts)))
    command.append("--task_index={0}".format(task_index)) 

    print ("executing command {}".format(command))
    run_training(command)


    #the above should output the command like below

    # desginating learner with ID 1 as ps and everyone else as worker
    #python mnist_dist.py \
    # --ps_hosts=ps0.example.com:2222,ps1.example.com:2222 \
    # --worker_hosts=worker0.example.com:2222,worker1.example.com:2222 \
    # --job_name=ps --task_index=0
