import sys
import os
import grpc

if sys.version_info[0] < 3:
    from training_data_service_client import training_data_pb2_grpc as td
else:
    from .training_data_pb2 import training_data_pb2_grpc as td

def get_connection():
    with open('./certs/server.crt') as f:
        certificate = f.read()

    credentials = grpc.ssl_channel_credentials(root_certificates=certificate)

    isTLSEnabled = True
    isLocal = False

    if isLocal:
        hosturl = '127.0.0.1'
        port = '30015'
    else:
        training_data_namespace = os.environ["TRAINING_DATA_NAMESPACE"]
        hosturl = "ffdl-trainingdata.%s.svc.cluster.local" % training_data_namespace
        port = '80'
        # hosturl = "10.177.1.186"
        # port = '80'

    hosturl = '{}:{}'.format(hosturl, port)

    print("hosturl: "+hosturl)
    sys.stdout.flush()

    if isTLSEnabled:
        channel = grpc.secure_channel(hosturl, credentials, options=(('grpc.ssl_target_name_override', 'dlaas.ibm.com',),))
    else:
        channel = grpc.insecure_channel(hosturl)

    tdClient = td.TrainingDataStub(channel)

    return tdClient

