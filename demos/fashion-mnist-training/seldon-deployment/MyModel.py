from keras.models import load_model
import numpy
import boto3
import botocore
import os

class MyModel(object):

    def __init__(self):
        # Initialize variables
        training_id = os.environ.get("TRAINING_ID")
        endpoint_url = os.environ.get("BUCKET_ENDPOINT_URL")
        bucket_name = os.environ.get("BUCKET_NAME")
        bucket_key = os.environ.get("BUCKET_KEY")
        bucket_secret = os.environ.get("BUCKET_SECRET")
        model_file_name = os.environ.get("MODEL_FILE_NAME", "model.h5")
        print("Training id:{} endpoint URL:{} key:{} secret:{} bucket name:{}".format(training_id,endpoint_url,bucket_key,bucket_secret,bucket_name))

        # Define S3 resource and download the model file
        client = boto3.resource(
            's3',
            endpoint_url=endpoint_url,
            aws_access_key_id=bucket_key,
            aws_secret_access_key=bucket_secret,
        )

        KEY = training_id + '/' + model_file_name
        model_path = 'model.h5'

        try:
            client.Bucket(bucket_name).download_file(KEY, model_path)
        except botocore.exceptions.ClientError as e:
            if e.response['Error']['Code'] == "404":
                print("The object does not exist.")
            else:
                raise

        # Replace with path of trained model
        self.model = load_model(model_path)

    def predict(self,X,features_names):
        X = numpy.array(X)
        return self.model.predict(X)
