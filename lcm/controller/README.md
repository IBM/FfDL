<!--
{% comment %}
Copyright 2017-2018 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
{% endcomment %}
-->

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


