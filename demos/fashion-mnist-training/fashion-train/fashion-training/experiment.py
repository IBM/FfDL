'''
Classification of Fashion MNIST images using a convolutional model written in Keras.
This example is using the Fashion MNIST database of clothing images provided by Zalando Research.
https://github.com/zalandoresearch/fashion-mnist

Author: IBM Watson
'''

import argparse
import gzip
import keras
from keras.callbacks import TensorBoard
from keras.layers import Dense, Dropout, Flatten
from keras.layers import Conv2D, MaxPooling2D
from keras.layers.normalization import BatchNormalization
from keras.models import Sequential
from keras.utils import to_categorical
import os
import numpy as np
import pandas as pd
from sklearn.model_selection import train_test_split
import sys
import time

epochs = 30
batch_size = 128
dropout = 0.4

# Add data dir to file path
data_dir = os.environ["DATA_DIR"]
train_images_file = os.path.join(data_dir, 'train-images-idx3-ubyte.gz')
train_labels_file = os.path.join(data_dir, 'train-labels-idx1-ubyte.gz')
test_images_file = os.path.join(data_dir, 't10k-images-idx3-ubyte.gz')
test_labels_file = os.path.join(data_dir, 't10k-labels-idx1-ubyte.gz')


# Load data in MNIST format
with gzip.open(train_labels_file, 'rb') as lbpath:
    y_train = np.frombuffer(lbpath.read(), dtype=np.uint8,
                           offset=8)

with gzip.open(train_images_file, 'rb') as imgpath:
    X_train = np.frombuffer(imgpath.read(), dtype=np.uint8,
                           offset=16).reshape(len(y_train), 784)

with gzip.open(test_labels_file, 'rb') as lbpath:
    y_test = np.frombuffer(lbpath.read(), dtype=np.uint8,
                           offset=8)

with gzip.open(test_images_file, 'rb') as imgpath:
    X_test = np.frombuffer(imgpath.read(), dtype=np.uint8,
                           offset=16).reshape(len(y_test), 784)

# Split a validation set off the train set
split = int(len(y_train) * .9)-1
X_train, X_val = X_train[:split], X_train[split:]
y_train, y_val = y_train[:split], y_train[split:]

y_train = to_categorical(y_train)
y_test = to_categorical(y_test)
y_val = to_categorical(y_val)

# Reshape to correct format for conv2d input
img_rows, img_cols = 28, 28
input_shape = (img_rows, img_cols, 1)

X_train = X_train.reshape(X_train.shape[0], img_rows, img_cols, 1)
X_test = X_test.reshape(X_test.shape[0], img_rows, img_cols, 1)
X_val = X_val.reshape(X_val.shape[0], img_rows, img_cols, 1)

X_train = X_train.astype('float32')
X_test = X_test.astype('float32')

X_train /= 255
X_test /= 255

model = Sequential()
model.add(Conv2D(32, kernel_size=(3, 3),
                 activation='relu',
                 kernel_initializer='he_normal',
                 input_shape=input_shape))
model.add(MaxPooling2D((2, 2)))
model.add(Dropout(dropout))
model.add(Conv2D(64, (3, 3), activation='relu'))
model.add(MaxPooling2D(pool_size=(2, 2)))
model.add(Dropout(dropout))
model.add(Conv2D(128, (3, 3), activation='relu'))
model.add(MaxPooling2D(pool_size=(2, 2)))
model.add(Dropout(dropout))
model.add(Flatten())
model.add(Dense(128, activation='relu'))
model.add(Dropout(dropout))

num_classes = 10
model.add(Dense(num_classes, activation='softmax'))

model.compile(loss='categorical_crossentropy',
              optimizer="adam",
              metrics=['accuracy'])

model.summary()

start_time = time.time()

tb_directory = os.environ["JOB_STATE_DIR"]+"/logs/tb/test"
tensorboard = TensorBoard(log_dir=tb_directory)
history = model.fit(X_train, y_train,
                    batch_size=batch_size,
                    epochs=epochs,
                    verbose=1,
                    validation_data=(X_val, y_val),
                    callbacks=[tensorboard])
score = model.evaluate(X_test, y_test, verbose=0)

end_time = time.time()
minutes, seconds = divmod(end_time-start_time, 60)
print("Total train time: {:0>2}:{:05.2f}".format(int(minutes),seconds))

print('Final train accuracy:      %.4f' % history.history['acc'][-1])
print('Final train loss: %.4f' % history.history['loss'][-1])
print('Final validation accuracy: %.4f' % history.history['val_acc'][-1])
print('Final validation loss: %.4f' % history.history['val_loss'][-1])
print('Final test accuracy:       %.4f' %  score[1])
print('Final test loss: %.4f' % score[0])

model_path = os.path.join(os.environ["RESULT_DIR"], "model.h5")
print("\nSaving model to: %s" % model_path)
model.save(model_path)
