'''
A Convolutional Network implementation example using TensorFlow library.
This example is using the MNIST database of handwritten digits
(http://yann.lecun.com/exdb/mnist/)

Author: Aymeric Damien
Project: https://github.com/aymericdamien/TensorFlow-Examples/
'''

import tensorflow as tf
import input_data
import sys
import os
import itertools
import re
import time
from random import randint

train_images_file = ""
train_labels_file = ""
test_images_file = ""
test_labels_file = ""

# Parameters
learning_rate = 0.001
training_iters = 200000
batch_size = 128
display_step = 10

model_path = os.environ["RESULT_DIR"] + "/model.ckpt"

# This helps distinguish instances when the training job is restarted.
instance_id = randint(0, 9999)

def main(argv):

    if len(argv) < 12:
        sys.exit("Not enough arguments provided.")

    global train_images_file, train_labels_file, test_images_file, test_labels_file, learning_rate, training_iters

    i = 1
    while i <= 12:
        arg = str(argv[i])
        if arg == "--trainImagesFile":
            train_images_file = str(argv[i+1])
        elif arg == "--trainLabelsFile":
            train_labels_file = str(argv[i+1])
        elif arg == "--testImagesFile":
            test_images_file = str(argv[i+1])
        elif arg == "--testLabelsFile":
            test_labels_file = str(argv[i+1])
        elif arg == "--learningRate":
            learning_rate = float(argv[i+1])
        elif arg =="--trainingIters":
            training_iters = int(argv[i+1])
        i += 2

if __name__ == "__main__":
    main(sys.argv)

# Import MINST data
mnist = input_data.read_data_sets(train_images_file,
    train_labels_file, test_images_file, test_labels_file, one_hot=True)

# Network Parameters
n_input = 784 # MNIST data input (img shape: 28*28)
n_classes = 10 # MNIST total classes (0-9 digits)
dropout = 0.75 # Dropout, probability to keep units

# tf Graph input
x = tf.placeholder(tf.float32, [None, n_input])
y = tf.placeholder(tf.float32, [None, n_classes])
keep_prob = tf.placeholder(tf.float32) #dropout (keep probability)


# Create some wrappers for simplicity
def conv2d(x, W, b, strides=1):
    # Conv2D wrapper, with bias and relu activation
    x = tf.nn.conv2d(x, W, strides=[1, strides, strides, 1], padding='SAME')
    x = tf.nn.bias_add(x, b)
    return tf.nn.relu(x)


def maxpool2d(x, k=2):
    # MaxPool2D wrapper
    return tf.nn.max_pool(x, ksize=[1, k, k, 1], strides=[1, k, k, 1],
                          padding='SAME')


# Create model
def conv_net(x, weights, biases, dropout):
    # Reshape input picture
    x = tf.reshape(x, shape=[-1, 28, 28, 1])

    # Convolution Layer
    conv1 = conv2d(x, weights['wc1'], biases['bc1'])
    # Max Pooling (down-sampling)
    conv1 = maxpool2d(conv1, k=2)

    # Convolution Layer
    conv2 = conv2d(conv1, weights['wc2'], biases['bc2'])
    # Max Pooling (down-sampling)
    conv2 = maxpool2d(conv2, k=2)

    # Fully connected layer
    # Reshape conv2 output to fit fully connected layer input
    fc1 = tf.reshape(conv2, [-1, weights['wd1'].get_shape().as_list()[0]])
    fc1 = tf.add(tf.matmul(fc1, weights['wd1']), biases['bd1'])
    fc1 = tf.nn.relu(fc1)
    # Apply Dropout
    fc1 = tf.nn.dropout(fc1, dropout)

    # Output, class prediction
    out = tf.add(tf.matmul(fc1, weights['out']), biases['out'])
    return out

# Store layers weight & bias
weights = {
    # 5x5 conv, 1 input, 32 outputs
    'wc1': tf.Variable(tf.random_normal([5, 5, 1, 32])),
    # 5x5 conv, 32 inputs, 64 outputs
    'wc2': tf.Variable(tf.random_normal([5, 5, 32, 64])),
    # fully connected, 7*7*64 inputs, 1024 outputs
    'wd1': tf.Variable(tf.random_normal([7*7*64, 1024])),
    # 1024 inputs, 10 outputs (class prediction)
    'out': tf.Variable(tf.random_normal([1024, n_classes]))
}

biases = {
    'bc1': tf.Variable(tf.random_normal([32])),
    'bc2': tf.Variable(tf.random_normal([64])),
    'bd1': tf.Variable(tf.random_normal([1024])),
    'out': tf.Variable(tf.random_normal([n_classes]))
}

# Construct model
pred = conv_net(x, weights, biases, keep_prob)

# Define loss and optimizer
cost = tf.reduce_mean(tf.nn.softmax_cross_entropy_with_logits(logits=pred, labels=y))
optimizer = tf.train.AdamOptimizer(learning_rate=learning_rate).minimize(cost)

# Evaluate model
correct_pred = tf.equal(tf.argmax(pred, 1), tf.argmax(y, 1))
accuracy = tf.reduce_mean(tf.cast(correct_pred, tf.float32))

# Initializing the variables
init = tf.global_variables_initializer()

saver = tf.train.Saver(max_to_keep=1)

# Assuming the checkpoint filename is "xyz-nnn", return nnn (as an int).
def get_step_from_checkpoint_name(filename):
    m = re.match(".*-([0-9]+)$", filename)
    if m is not None:
        return int(m.group(1))
    else:
        return None

# Restore from a checkpoint if possible, and return the step count of the checkpoint.
def restore_from_checkpoint():
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

def save_checkpoint(step):
    path = saver.save(sess, model_path, global_step=step)
    print("Saved checkpoint %s" % path)


# Launch the graph
with tf.Session() as sess:
    sess.run(init)
    step = restore_from_checkpoint()
    if step is None:
        step = 1
    # Keep training until reach max iterations
    while step * batch_size < training_iters:
        batch_x, batch_y = mnist.train.next_batch(batch_size)
        # Run optimization op (backprop)
        sess.run(optimizer, feed_dict={x: batch_x, y: batch_y,
                                       keep_prob: dropout})
        if step % display_step == 0:
            save_checkpoint(step)
            # Calculate batch loss and accuracy
            loss, acc = sess.run([cost, accuracy], feed_dict={x: batch_x,
                                                              y: batch_y,
                                                              keep_prob: 1.})
            print("Time " + "{:.4f}".format(time.time()) + \
                  ", instance " + str(instance_id) + \
                  ", Iter " + str(step * batch_size) + \
                  ", Minibatch Loss= " + "{:.6f}".format(loss) + \
                  ", Training Accuracy= " + "{:.5f}".format(acc))
            sys.stdout.flush()
        step += 1
    print("Optimization Finished!")

    # Calculate accuracy for 256 mnist test images
    print("Testing Accuracy:", \
        sess.run(accuracy, feed_dict={x: mnist.test.images[:256],
                                      y: mnist.test.labels[:256],
                                      keep_prob: 1.}))

    save_path = saver.save(sess, model_path)
    print("Model saved in file: %s" % save_path)
    sys.stdout.flush()
