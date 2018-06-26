#-----------------------------------------------------------------------
# This information contains sample code provided in source code form.
# You may copy, modify, and distribute these sample programs in any 
# form without payment to IBM for the purposes of developing, using,
# marketing or distributing application programs conforming to the     
# application programming interface for the operating platform for     
# which the sample code is written. Notwithstanding anything to the 
# contrary, IBM PROVIDES THE SAMPLE SOURCE CODE ON AN 'AS IS' BASIS 
# AND IBM DISCLAIMS ALL WARRANTIES, EXPRESS OR IMPLIED, INCLUDING,     
# BUT NOT LIMITED TO, ANY IMPLIED WARRANTIES OR CONDITIONS OF          
# MERCHANTABILITY, SATISFACTORY QUALITY, FITNESS FOR A PARTICULAR      
# PURPOSE, TITLE, AND ANY WARRANTY OR CONDITION OF NON-INFRINGEMENT.
# IBM SHALL NOT BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,     
# SPECIAL, EXEMPLARY OR ECONOMIC CONSEQUENTIAL DAMAGES ARISING OUT     
# OF THE USE OR OPERATION OF THE SAMPLE SOURCE CODE. IBM SHALL NOT     
# BE LIABLE FOR LOSS OF, OR DAMAGE TO, DATA, OR FOR LOST PROFITS,     
# BUSINESS REVENUE, GOODWILL, OR ANTICIPATED SAVINGS. IBM HAS NO     
# OBLIGATION TO PROVIDE MAINTENANCE, SUPPORT, UPDATES, ENHANCEMENTS 
# OR MODIFICATIONS TO THE SAMPLE SOURCE CODE.                        
#-------------------------------------------------------------------

import os
import sys

import tensorflow as tf
from tensorflow.examples.tutorials.mnist import input_data

flags = tf.app.flags
flags.DEFINE_string("data_dir", os.getenv("DATA_DIR", "/tmp/mnist-data"), "Directory for storing mnist data")
flags.DEFINE_string("model_dir", os.getenv("RESULT_DIR", "."), "Directory for storing trained model")
flags.DEFINE_string("ps_hosts", "", "Comma-separated list of hostname:port pairs")
flags.DEFINE_string("worker_hosts", "", "Comma-separated list of hostname:port pairs")
flags.DEFINE_string("job_name", "", "One of 'ps', 'worker'")
flags.DEFINE_integer("task_index", 0, "Index of task within the job")
flags.DEFINE_integer("batch_size", 100, "Index of task within the job")
flags.DEFINE_integer('model_version', 1, 'version number of the model.')
FLAGS = flags.FLAGS


def main(_):
    ps_hosts = FLAGS.ps_hosts.split(",")

    worker_hosts = FLAGS.worker_hosts.split(",")
    cluster = tf.train.ClusterSpec({"ps": ps_hosts, "worker": worker_hosts})
    server = tf.train.Server(cluster, job_name=FLAGS.job_name, task_index=FLAGS.task_index)

    if FLAGS.job_name == "ps":
        server.join()
    elif FLAGS.job_name == "worker":
        train(server, cluster)


def train(server, cluster):
    print('Training model...')
    with tf.device(tf.train.replica_device_setter(worker_device="/job:worker/task:%d" % FLAGS.task_index, cluster=cluster)):
        mnist = input_data.read_data_sets(FLAGS.data_dir, one_hot=True)
        serialized_tf_example = tf.placeholder(tf.string, name='tf_example')
        feature_configs = {'x': tf.FixedLenFeature(shape=[784], dtype=tf.float32), }
        tf_example = tf.parse_example(serialized_tf_example, feature_configs)

        x = tf.identity(tf_example['x'], name='x')  # use tf.identity() to assign name
        y_ = tf.placeholder('float', shape=[None, 10])

        w = tf.Variable(tf.zeros([784, 10]))
        b = tf.Variable(tf.zeros([10]))

        y = tf.nn.softmax(tf.matmul(x, w) + b, name='y')
        cross_entropy = -tf.reduce_sum(y_ * tf.log(y))

        global_step = tf.Variable(0)
        train_step = tf.train.GradientDescentOptimizer(0.01).minimize(cross_entropy, global_step=global_step)
        values, indices = tf.nn.top_k(y, 10)

        # Use xrange in python2, range in python3
        try:
          xrange
        except NameError:
          xrange = range
        prediction_classes = tf.contrib.lookup.index_to_string(tf.to_int64(indices), mapping=tf.constant([str(i) for i in xrange(10)]))
        correct_prediction = tf.equal(tf.argmax(y, 1), tf.argmax(y_, 1))
        accuracy = tf.reduce_mean(tf.cast(correct_prediction, 'float'))

        summary_op = tf.summary.merge_all()
        init_op = tf.global_variables_initializer()
        saver = tf.train.Saver()

        sv = tf.train.Supervisor(is_chief=(FLAGS.task_index == 0), logdir="train_logs", init_op=init_op,
                                 summary_op=summary_op, saver=saver, global_step=global_step, save_model_secs=600)

        with sv.managed_session(server.target) as sess:
            step = 0

            while not sv.should_stop() and step < 1000:
                batch_xs, batch_ys = mnist.train.next_batch(FLAGS.batch_size)
                train_feed = {x: batch_xs, y_: batch_ys}

                _, step = sess.run([train_step, global_step], feed_dict=train_feed)

                if step % 1000 == 0:
                    print("global step: {} , accuracy:{}".format(step, sess.run(accuracy, feed_dict=train_feed)))

            print('training accuracy %g' % sess.run(accuracy, feed_dict={x: mnist.test.images, y_: mnist.test.labels}))
            print('Done training!')
            if sv.is_chief:
                sess.graph._unsafe_unfinalize()
                export_path_base = str(FLAGS.model_dir)
                export_path = os.path.join(
                    tf.compat.as_bytes(export_path_base),
                    tf.compat.as_bytes(str(FLAGS.model_version)))
                print('Exporting trained model to', export_path)
                builder = tf.saved_model.builder.SavedModelBuilder(export_path)

                # Build the signature_def_map.
                classification_inputs = tf.saved_model.utils.build_tensor_info(serialized_tf_example)
                classification_outputs_classes = tf.saved_model.utils.build_tensor_info(prediction_classes)
                classification_outputs_scores = tf.saved_model.utils.build_tensor_info(values)

                classification_signature = tf.saved_model.signature_def_utils.build_signature_def(
                    inputs={tf.saved_model.signature_constants.CLASSIFY_INPUTS: classification_inputs},
                    outputs={
                        tf.saved_model.signature_constants.CLASSIFY_OUTPUT_CLASSES: classification_outputs_classes,
                        tf.saved_model.signature_constants.CLASSIFY_OUTPUT_SCORES: classification_outputs_scores
                    },
                    method_name=tf.saved_model.signature_constants.CLASSIFY_METHOD_NAME)

                tensor_info_x = tf.saved_model.utils.build_tensor_info(x)
                tensor_info_y = tf.saved_model.utils.build_tensor_info(y)

                prediction_signature = tf.saved_model.signature_def_utils.build_signature_def(
                    inputs={'images': tensor_info_x},
                    outputs={'scores': tensor_info_y},
                    method_name=tf.saved_model.signature_constants.PREDICT_METHOD_NAME)

                legacy_init_op = tf.group(tf.tables_initializer(), name='legacy_init_op')

                builder.add_meta_graph_and_variables(
                    sess, [tf.saved_model.tag_constants.SERVING],
                    signature_def_map={
                        'predict_images': prediction_signature,
                        tf.saved_model.signature_constants.DEFAULT_SERVING_SIGNATURE_DEF_KEY: classification_signature,
                    },
                    legacy_init_op=legacy_init_op,
                    clear_devices=True)

                builder.save()
                print('Done exporting!')

if __name__ == '__main__':
    tf.app.run()
