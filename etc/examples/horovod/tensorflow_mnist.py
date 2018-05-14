#!/usr/bin/env python

#example picked from https://raw.githubusercontent.com/uber/horovod/master/examples/tensorflow_mnist.py
import os, sys
import tensorflow as tf
import horovod.tensorflow as hvd
layers = tf.contrib.layers
learn = tf.contrib.learn

tf.logging.set_verbosity(tf.logging.INFO)

checkpoint_path = os.getenv("RESULT_DIR")
training_data_dir = os.getenv("DATA_DIR")
learner_id = os.getenv("LEARNER_ID")

print("checkpint path is {} and training_data_dir is {} and learner_id {}".format( checkpoint_path, training_data_dir, learner_id))
if checkpoint_path == None:
    exit(1)

if training_data_dir == None:
    exit(1)

if learner_id == None:
    exit(1)

training_data_dir = training_data_dir+"/mnist/"


def conv_model(feature, target, mode):
    """2-layer convolution model."""
    # Convert the target to a one-hot tensor of shape (batch_size, 10) and
    # with a on-value of 1 for each one-hot vector of length 10.
    target = tf.one_hot(tf.cast(target, tf.int32), 10, 1, 0)

    # Reshape feature to 4d tensor with 2nd and 3rd dimensions being
    # image width and height final dimension being the number of color channels.
    feature = tf.reshape(feature, [-1, 28, 28, 1])

    # First conv layer will compute 32 features for each 5x5 patch
    with tf.variable_scope('conv_layer1'):
        h_conv1 = layers.conv2d(
            feature, 32, kernel_size=[5, 5], activation_fn=tf.nn.relu)
        h_pool1 = tf.nn.max_pool(
            h_conv1, ksize=[1, 2, 2, 1], strides=[1, 2, 2, 1], padding='SAME')

    # Second conv layer will compute 64 features for each 5x5 patch.
    with tf.variable_scope('conv_layer2'):
        h_conv2 = layers.conv2d(
            h_pool1, 64, kernel_size=[5, 5], activation_fn=tf.nn.relu)
        h_pool2 = tf.nn.max_pool(
            h_conv2, ksize=[1, 2, 2, 1], strides=[1, 2, 2, 1], padding='SAME')
        # reshape tensor into a batch of vectors
        h_pool2_flat = tf.reshape(h_pool2, [-1, 7 * 7 * 64])

    # Densely connected layer with 1024 neurons.
    h_fc1 = layers.dropout(
        layers.fully_connected(
            h_pool2_flat, 1024, activation_fn=tf.nn.relu),
        keep_prob=0.5,
        is_training=mode == tf.contrib.learn.ModeKeys.TRAIN)

    # Compute logits (1 per class) and compute loss.
    logits = layers.fully_connected(h_fc1, 10, activation_fn=None)
    loss = tf.losses.softmax_cross_entropy(target, logits)

    return tf.argmax(logits, 1), loss


def main(argv):
    # Initialize Horovod.
    hvd.init()
    # Download and load MNIST dataset.
    mnist = learn.datasets.mnist.read_data_sets(training_data_dir)

    # Build model...
    with tf.name_scope('input'):
        image = tf.placeholder(tf.float32, [None, 784], name='image')
        label = tf.placeholder(tf.float32, [None], name='label')
    predict, loss = conv_model(image, label, tf.contrib.learn.ModeKeys.TRAIN)

    opt = tf.train.RMSPropOptimizer(0.01)

    # Add Horovod Distributed Optimizer.
    opt = hvd.DistributedOptimizer(opt)

    global_step = tf.contrib.framework.get_or_create_global_step()
    train_op = opt.minimize(loss, global_step=global_step)

    # BroadcastGlobalVariablesHook broadcasts initial variable states from rank 0
    # to all other processes. This is necessary to ensure consistent initialization
    # of all workers when training is started with random weights or restored
    # from a checkpoint.
    hooks = [hvd.BroadcastGlobalVariablesHook(0),
             tf.train.StopAtStepHook(last_step=100),
             tf.train.LoggingTensorHook(tensors={'step': global_step, 'loss': loss},
                                        every_n_iter=10),
             ]

    # Pin GPU to be used to process local rank (one GPU per process)
    config = tf.ConfigProto()
    config.gpu_options.allow_growth = True
    config.gpu_options.visible_device_list = str(hvd.local_rank())

    # Save checkpoints only on worker 0 to prevent other workers from corrupting them.
    checkpoint_dir = checkpoint_path if hvd.rank() == 0 else None

    # The MonitoredTrainingSession takes care of session initialization,
    # restoring from a checkpoint, saving to a checkpoint, and closing when done
    # or an error occurs.
    with tf.train.MonitoredTrainingSession(checkpoint_dir=checkpoint_dir,
                                           hooks=hooks,
                                           config=config) as mon_sess:
        while not mon_sess.should_stop():
            # Run a training step synchronously.
            image_, label_ = mnist.train.next_batch(100)
            mon_sess.run(train_op, feed_dict={image: image_, label: label_})


if __name__ == "__main__":
    tf.app.run()
