import onnx
import os
from onnx_tf.backend import prepare

class MyModel(object):

    def __init__(self):
        # Initialize variables
        training_id = os.environ.get("TRAINING_ID")
        model_file_name = os.environ.get("MODEL_FILE_NAME")
        mountPath = os.environ.get("MOUNT_PATH")

        # Replace with path of trained model
        model_path = mountPath + '/' + training_id + '/' + model_file_name

        self.model = onnx.load(model_path)
        self.tf_rep = prepare(self.model)

    def predict(self,X,features_names):
        return self.tf_rep.run(X)


model = onnx.load(model_path)
tf_output = onnx_tf.backend.prepare(model).run(X)
caffe2_output = caffe2.python.onnx.backend.run_model(model, [img])
