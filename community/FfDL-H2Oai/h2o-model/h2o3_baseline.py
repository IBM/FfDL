import h2o
from h2o.automl import H2OAutoML
import os
import sys
import socket
import time


train_data_file = None
memory = None
target = None


def parse_args(argv):
    if len(argv) < 6:
        sys.exit("Not enough arguments provided")

    global train_data_file, memory, target

    i = 1
    while i <= 6:
        arg = str(argv[i])
        if arg == "--trainDataFile":
            train_data_file = str(argv[i+1])
        elif arg == "--memory":
            memory = str(argv[i+1])
        elif arg == "--target":
            target = str(argv[i+1])
        i += 2


if __name__ == "__main__":
    parse_args(sys.argv)

h2o.init(ip=socket.gethostbyname(socket.gethostname()), port="54321", start_h2o=False)

train = h2o.import_file(train_data_file)

x = train.columns
y = target
x.remove(y)

train[y] = train[y].asfactor()

aml = H2OAutoML(max_runtime_secs=60)
aml.train(x=x, y=y, training_frame=train)

lb = aml.leaderboard
print(lb)

save_path = os.environ["RESULT_DIR"]
model_path = h2o.save_model(model=aml.leader, path=save_path)
print(model_path)

h2o.cluster().shutdown()
