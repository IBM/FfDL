#
# Copyright 2017-2018 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import tensorflow as tf
import boto3
import botocore
import tarfile
import os

class TFMnist(object):
    def __init__(self):
        training_id = os.environ.get("TRAINING_ID")
        endpoint_url = os.environ.get("BUCKET_ENDPOINT_URL")
        bucket_name = os.environ.get("BUCKET_NAME")
        bucket_key = os.environ.get("BUCKET_KEY")
        bucket_secret = os.environ.get("BUCKET_SECRET")
        print("Training id:{} endpoint URL:{} key:{} secret:{}".format(training_id,endpoint_url,bucket_key,bucket_secret))
        
        self.class_names = ["class:{}".format(str(i)) for i in range(10)]
        #self.class_names = ["prediction"]

        # Define S3 resource and download the model file
        client = boto3.resource(
            's3',
            endpoint_url=endpoint_url,
            aws_access_key_id=bucket_key,
            aws_secret_access_key=bucket_secret,
        )

        KEY = training_id + '/saved_model.tar.gz' # replace with your object key

        try:
            client.Bucket(bucket_name).download_file(KEY, 'saved_model.tar.gz')
        except botocore.exceptions.ClientError as e:
            if e.response['Error']['Code'] == "404":
                print("The object does not exist.")
            else:
                raise

        # Untar model file
        tar = tarfile.open("saved_model.tar.gz")
        tar.extractall()
        tar.close()

        # Load the model into tf session and run predictions.
        self.sess = tf.Session()
        #tf.Session(graph=tf.Graph()) as sess:
        # Load saved model into tf session
        tf.saved_model.loader.load(self.sess, [tf.saved_model.tag_constants.SERVING], "./")
        graph = tf.get_default_graph()
        self.input = graph.get_tensor_by_name("x_input:0")
        #self.predictor = graph.get_tensor_by_name("predictor:0")
        self.output = graph.get_tensor_by_name("y_output:0")
        self.keep_prob = tf.placeholder(tf.float32)

    def predict(self,X,feature_names):
        predictions = self.sess.run(self.output, feed_dict = {self.input:X, self.keep_prob:1.0})
        return predictions


    

