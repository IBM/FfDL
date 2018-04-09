# Copyright 2015 The TensorFlow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the 'License');
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an 'AS IS' BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# ==============================================================================
"""A simple MNIST classifier which displays summaries in TensorBoard.

 This is an unimpressive MNIST model, but it is a good example of using
tf.name_scope to make a graph legible in the TensorBoard graph explorer, and of
naming summary tags so that they are grouped meaningfully in TensorBoard.

It demonstrates the functionality of every TensorBoard dashboard.
"""
from __future__ import absolute_import
from __future__ import division
from __future__ import print_function

import argparse
import sys
import os

import tensorflow as tf

import input_data

# from tensorflow.examples.tutorials.mnist import input_data

FLAGS = None

tf.logging.set_verbosity(tf.logging.INFO)

global train_images_file, train_labels_file, test_images_file
global test_labels_file, learning_rate
# global saver

# Assuming the checkpoint filename is "xyz-nnn", return nnn (as an int).
def get_step_from_checkpoint_name(filename):
    m = re.match(".*-([0-9]+)$", filename)
    if m is not None:
        return int(m.group(1))
    else:
        return None

# Restore from a checkpoint if possible, and return the step count of the checkpoint.
def restore_from_checkpoint(saver, sess):
    cs = tf.train.get_checkpoint_state(os.environ["RESULT_DIR"])
    checkpoints = []
    if cs is not None:
        checkpoints = itertools.chain([cs.model_checkpoint_path], reversed(cs.all_model_checkpoint_paths))
    step = None
    for c in checkpoints:
        try:
            saver.restore(sess, c)
            print("Restored from checkpoint: %s" % c)
            step = get_step_from_checkpoint_name(c)
            break
        except tf.errors.NotFoundError:
            print("Warning: checkpoint not found: %s" % c)
        except tf.errors.DataLossError:
            print("Warning: checkpoint corrupted: %s" % c)
    return step

def save_checkpoint(step, saver, sess, model_path):
    path = saver.save(sess, model_path, global_step=step)
    print("Saved checkpoint %s" % path)

def train():

    # Import data
    #   mnist = input_data.read_data_sets(FLAGS.data_dir,
    #                                     one_hot=True,
    #                                     fake_data=FLAGS.fake_data)
    mnist = input_data.read_data_sets(FLAGS.train_images_file,
                                      FLAGS.train_labels_file,
                                      FLAGS.test_images_file,
                                      FLAGS.test_labels_file,
                                      one_hot=True)

    sess = tf.InteractiveSession()

    # Create a multilayer model.

    # Input placeholders
    with tf.name_scope('input'):
        x = tf.placeholder(tf.float32, [None, 784], name='x-input')
        y_ = tf.placeholder(tf.float32, [None, 10], name='y-input')

    with tf.name_scope('input_reshape'):
        image_shaped_input = tf.reshape(x, [-1, 28, 28, 1])
        tf.summary.image('input', image_shaped_input, 10)

    # We can't initialize these variables to 0 - the network will get stuck.
    def weight_variable(shape):
        """Create a weight variable with appropriate initialization."""
        initial = tf.truncated_normal(shape, stddev=0.1)
        return tf.Variable(initial)

    def bias_variable(shape):
        """Create a bias variable with appropriate initialization."""
        initial = tf.constant(0.1, shape=shape)
        return tf.Variable(initial)

    def variable_summaries(var):
        """Attach a lot of summaries to a Tensor (for TensorBoard visualization)."""
        with tf.name_scope('summaries'):
            mean = tf.reduce_mean(var)
            tf.summary.scalar('mean', mean)
            with tf.name_scope('stddev'):
                stddev = tf.sqrt(tf.reduce_mean(tf.square(var - mean)))
            tf.summary.scalar('stddev', stddev)
            tf.summary.scalar('max', tf.reduce_max(var))
            tf.summary.scalar('min', tf.reduce_min(var))
            tf.summary.histogram('histogram', var)

    def nn_layer(input_tensor, input_dim, output_dim, layer_name, act=tf.nn.relu):
        """Reusable code for making a simple neural net layer.

        It does a matrix multiply, bias add, and then uses relu to nonlinearize.
        It also sets up name scoping so that the resultant graph is easy to read,
        and adds a number of summary ops.
        """
        # Adding a name scope ensures logical grouping of the layers in the graph.
        with tf.name_scope(layer_name):
            # This Variable will hold the state of the weights for the layer
            with tf.name_scope('weights'):
                weights = weight_variable([input_dim, output_dim])
                variable_summaries(weights)
            with tf.name_scope('biases'):
                biases = bias_variable([output_dim])
                variable_summaries(biases)
            with tf.name_scope('Wx_plus_b'):
                preactivate = tf.matmul(input_tensor, weights) + biases
                tf.summary.histogram('pre_activations', preactivate)
            activations = act(preactivate, name='activation')
            tf.summary.histogram('activations', activations)
            return activations

    hidden1 = nn_layer(x, 784, 500, 'layer1')

    with tf.name_scope('dropout'):
        keep_prob = tf.placeholder(tf.float32)
        tf.summary.scalar('dropout_keep_probability', keep_prob)
        dropped = tf.nn.dropout(hidden1, keep_prob)

    # Do not apply softmax activation yet, see below.
    y = nn_layer(dropped, 500, 10, 'layer2', act=tf.identity)

    with tf.name_scope('cross_entropy'):
        # The raw formulation of cross-entropy,
        #
        # tf.reduce_mean(-tf.reduce_sum(y_ * tf.log(tf.softmax(y)),
        #                               reduction_indices=[1]))
        #
        # can be numerically unstable.
        #
        # So here we use tf.nn.softmax_cross_entropy_with_logits on the
        # raw outputs of the nn_layer above, and then average across
        # the batch.
        diff = tf.nn.softmax_cross_entropy_with_logits(labels=y_, logits=y)
        with tf.name_scope('total'):
            cross_entropy = tf.reduce_mean(diff)
    tf.summary.scalar('cross_entropy', cross_entropy)

    with tf.name_scope('train'):
        train_step = tf.train.AdamOptimizer(FLAGS.learning_rate).minimize(
            cross_entropy)

    with tf.name_scope('accuracy'):
        with tf.name_scope('correct_prediction'):
            correct_prediction = tf.equal(tf.argmax(y, 1), tf.argmax(y_, 1))
        with tf.name_scope('accuracy'):
            accuracy = tf.reduce_mean(tf.cast(correct_prediction, tf.float32))
    tf.summary.scalar('accuracy', accuracy)

    #     with tf.name_scope('loss'):
    #         # loss = tf.reduce_mean(tf.nn.softmax_cross_entropy_with_logits(logits=mnist.test.images, labels=mnist.test.labels))
    #         loss = tf.reduce_mean(tf.nn.softmax_cross_entropy_with_logits(logits=y_, labels=y))
    #         # loss = 10.5
    #     tf.summary.scalar('loss', loss)

    # Merge all the summaries and write them out to /tmp/tensorflow/mnist/logs/tb (by default)
    merged = tf.summary.merge_all()
    train_writer = tf.summary.FileWriter(FLAGS.tb_log_dir + '/train', sess.graph)
    test_writer = tf.summary.FileWriter(FLAGS.tb_log_dir + '/test')
    tf.global_variables_initializer().run()

    # Train the model, and also write summaries.
    # Every 10th step, measure test-set accuracy, and write test summaries
    # All other steps, run train_step on training data, & add training summaries

    def feed_dict(train):
        """Make a TensorFlow feed_dict: maps data onto Tensor placeholders."""
        if train or FLAGS.fake_data:
            xs, ys = mnist.train.next_batch(100, fake_data=FLAGS.fake_data)
            k = FLAGS.dropout
        else:
            xs, ys = mnist.test.images, mnist.test.labels
            k = 1.0
        return {x: xs, y_: ys, keep_prob: k}

    model_path = os.environ["RESULT_DIR"] + "/model.ckpt"

    print('Calling tf.train.Saver(...)')
    saver = tf.train.Saver(max_to_keep=1)

    step = restore_from_checkpoint(saver, sess)
    if step is None:
        step = 1

    for i in range(step-1, FLAGS.max_steps):
        if i % 10 == 0:  # Record summaries and test-set accuracy
            save_checkpoint(step, saver, sess, model_path)
            summary, acc = sess.run([merged, accuracy], feed_dict=feed_dict(False))
            test_writer.add_summary(summary, i)
            print('Accuracy at step %s: %s' % (i, acc))
        else:  # Record train set summaries, and train
            # Commnent the below code since CUPTI library is not Available on basic
            # TensorFlow images that use CUDA 9.
            #
            # if i % 100 == 99:  # Record execution stats
            #     run_options = tf.RunOptions(trace_level=tf.RunOptions.FULL_TRACE)
            #     run_metadata = tf.RunMetadata()
            #     # summary, _, loss_result = sess.run([merged, train_step, loss],
            #     summary, _ = sess.run([merged, train_step],
            #                           feed_dict=feed_dict(True),
            #                           options=run_options,
            #                           run_metadata=run_metadata)
            #     train_writer.add_run_metadata(run_metadata, 'step%03d' % i)
            #     train_writer.add_summary(summary, i)
            #     print('Adding run metadata for', i)
            # else:  # Record a summary
            summary, _ = sess.run([merged, train_step], feed_dict=feed_dict(True))
            train_writer.add_summary(summary, i)

    save_path = saver.save(sess, model_path)

    train_writer.close()
    test_writer.close()

def main(_):
    print('train_images_file: ', FLAGS.train_images_file)
    print('train_labels_file: ', FLAGS.train_labels_file)
    print('test_images_file: ', FLAGS.test_images_file)
    print('test_labels_file: ', FLAGS.test_labels_file)
    print('learning_rate: ', FLAGS.learning_rate)
    print('tb_log_dir: ', FLAGS.tb_log_dir)
    print('log_dir: ', FLAGS.log_dir)

    # if tf.gfile.Exists(FLAGS.log_dir):
    #    tf.gfile.DeleteRecursively(FLAGS.log_dir)
    tf.gfile.MakeDirs(FLAGS.log_dir)

    # if tf.gfile.Exists(FLAGS.tb_log_dir):
    #    tf.gfile.DeleteRecursively(FLAGS.tb_log_dir)
    tf.gfile.MakeDirs(FLAGS.tb_log_dir)

    train()


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--fake_data', nargs='?', const=True, type=bool,
                        default=False,
                        help='If true, uses fake data for unit testing.')
    parser.add_argument('--max_steps', type=int, default=1000,
                        help='Number of steps to run trainer.')
    parser.add_argument('--learning_rate', type=float, default=0.001,
                        help='Initial learning rate')
    parser.add_argument('--dropout', type=float, default=0.9,
                        help='Keep probability for training dropout.')
    #     parser.add_argument('--data_dir', type=str, default='/tmp/tensorflow/mnist/input_data',
    #                         help='Directory for storing input data')

    if 'LC_TEST_ENV' in os.environ:
        is_test_env = os.environ["LC_TEST_ENV"] == "local_test"
    else:
        is_test_env = None

    # TODO: we should be using an environment variable for job_directory
    if is_test_env:
        job_directory = "./job"
    else:
        job_directory = os.environ["JOB_STATE_DIR"]
        # job_directory = "/job"

    if is_test_env:
        if not os.path.exists(job_directory):
            os.makedirs(job_directory)
        logdir = job_directory + "/logs"
        if not os.path.exists(logdir):
            os.makedirs(logdir)

    log_directory = job_directory + "/logs"

    parser.add_argument('--log_dir', type=str, default=log_directory,
                        help='DLaaS log directory')
    parser.add_argument('--tb_log_dir', type=str, default=log_directory+'/tb',
                        help='Summaries log directory')

    parser.add_argument('--train_images_file', type=str, default='bad',
                        help='train_images_file')
    parser.add_argument('--train_labels_file', type=str, default='bad',
                        help='train_labels_file')
    parser.add_argument('--test_images_file', type=str, default='bad',
                        help='test_images_file')
    parser.add_argument('--test_labels_file', type=str, default='bad',
                        help='test_labels_file')

    FLAGS, unparsed = parser.parse_known_args()

    tf.app.run(main=main, argv=[sys.argv[0]] + unparsed)
