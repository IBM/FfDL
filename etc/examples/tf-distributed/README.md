This example runs mnist under TensorFlow in a distributed manner.  The code is written for any number of parameter servers and workers, 
which are passed in as arguments.  It uses the CPU only and should complete in roughly half a minute.

For example, to run with one parameter server and two workers, you would open three terminals and run the following commands (one per terminal):
```python3 mnist_dist.py --ps_hosts localhost:2222 --worker_hosts localhost:2223,localhost:2224 --job_name ps --task_index 0```
```python3 mnist_dist.py --ps_hosts localhost:2222 --worker_hosts localhost:2223,localhost:2224 --job_name worker --task_index 0```
```python3 mnist_dist.py --ps_hosts localhost:2222 --worker_hosts localhost:2223,localhost:2224 --job_name worker --task_index 1```

A few additional notes:
* This program will download the data locally if it's not already present.
* The order in which you start the processes doesn't matter, but the workers will not make progress until the parameter server is up.
* The workers run asynchronously, so if you never start the second worker, the first worker may run through all the data and the program will still complete.
* The parameter server process does not exit.  You have to kill it when the workers are done.
