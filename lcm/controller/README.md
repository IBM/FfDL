# Controller

This component implements a state machine to schedule the containers in a learner pod, and updates Zookeeper about the status of the job.

The current state of the state machine is in a file called `$JOB_STATE_DIR/current_state`. Files in the `$JOB_STATE_DIR`
are used to trigger the other containers in the pod, and to record the completion of the other containers.


# State machine

The transitions in the state machine are documented in the ./docs/state-machine.dot file.

To build a visual representation of the state machine:
```
$ make docs
```
The output is in the ./docs/state-machine.pdf file.


