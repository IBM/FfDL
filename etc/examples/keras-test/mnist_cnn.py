#-----------------------------------------------------------------------
#                                                                   
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
#                                                                   
#-------------------------------------------------------------------

'''Trains a simple convnet on the MNIST dataset.
Gets to 99.25% test accuracy after 12 epochs
(there is still a lot of margin for parameter tuning).
16 seconds per epoch on a GRID K520 GPU.
'''

from __future__ import print_function
import keras
import numpy as np
from keras.models import Sequential
from keras.layers import Dense, Dropout, Flatten
from keras.layers import Conv2D, MaxPooling2D
from keras import backend as K
import sys
import os

batch_size = 128
num_classes = 10
epochs = 1 # Only one epoch, to keep it quick

# input image dimensions
img_rows, img_cols = 28, 28

model_path = os.environ["RESULT_DIR"] + "/keras_mnist_cnn.hdf5"

def main(argv):
    if len(argv) < 2:
        sys.exit("Not enough arguments provided.")
        
    global image_path

    i = 1
    while i <= 2:
        arg = str(argv[i])
        if arg == "--mnistData":
            image_path = str(argv[i+1])
        i += 2

if __name__ == "__main__":
    main(sys.argv)

# the data, shuffled and split between train and test sets
f = np.load(image_path)
x_train = f['x_train']
y_train = f['y_train']
x_test = f['x_test']
y_test = f['y_test']
f.close()

if K.image_data_format() == 'channels_first':
    x_train = x_train.reshape(x_train.shape[0], 1, img_rows, img_cols)
    x_test = x_test.reshape(x_test.shape[0], 1, img_rows, img_cols)
    input_shape = (1, img_rows, img_cols)
else:
    x_train = x_train.reshape(x_train.shape[0], img_rows, img_cols, 1)
    x_test = x_test.reshape(x_test.shape[0], img_rows, img_cols, 1)
    input_shape = (img_rows, img_cols, 1)

x_train = x_train.astype('float32')
x_test = x_test.astype('float32')
x_train /= 255
x_test /= 255
print('x_train shape:', x_train.shape)
print(x_train.shape[0], 'train samples')
print(x_test.shape[0], 'test samples')

# convert class vectors to binary class matrices
y_train = keras.utils.to_categorical(y_train, num_classes)
y_test = keras.utils.to_categorical(y_test, num_classes)

model = Sequential()
model.add(Conv2D(32, kernel_size=(3, 3),
                 activation='relu',
                 input_shape=input_shape))
model.add(Conv2D(64, (3, 3), activation='relu'))
model.add(MaxPooling2D(pool_size=(2, 2)))
model.add(Dropout(0.25))
model.add(Flatten())
model.add(Dense(128, activation='relu'))
model.add(Dropout(0.5))
model.add(Dense(num_classes, activation='softmax'))

model.compile(loss=keras.losses.categorical_crossentropy,
              optimizer=keras.optimizers.Adadelta(),
              metrics=['accuracy'])

model.fit(x_train, y_train, batch_size=batch_size, epochs=epochs,
          verbose=1, validation_data=(x_test, y_test))
score = model.evaluate(x_test, y_test, verbose=0)
print('Test loss:', score[0])
print('Test accuracy:', score[1])

model.save(model_path)
print("Model saved in file: %s" % model_path)
