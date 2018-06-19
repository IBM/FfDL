#!/usr/bin/env python

#
# Copyright 2018 IBM Corp. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

import collections
import json
import logging
import os
import requests
import signal
import time
import threading
from tornado import httpserver, ioloop, web
from tornado.options import define, options, parse_command_line
import numpy
from PIL import Image
import io

modelEndpoint = os.environ.get("MODEL_ENDPOINT")
# Command Line Options
define("port", default=8088, help="Port the web app will run on")
# define("ml-endpoint", default="http://0.0.0.0:5000/predict",
#        help="The Image Caption Generator REST endpoint")
define("ml-endpoint", default=modelEndpoint,
       help="The Image Caption Generator REST endpoint")

# Setup Logging
logging.basicConfig(level=os.environ.get("LOGLEVEL", "INFO"),
                    format='%(levelname)s: %(message)s')

# Global variables
static_img_path = "static/img/images/"
temp_img_prefix = "MAX-"
image_captions = collections.OrderedDict()
VALID_EXT = ['png', 'jpg', 'jpeg', 'gif']
FASHION_LIST = ['T-shirt/top','Trouser','Pullover','Dress','Coat','Sandal','Shirt','Sneaker','Bag','Ankle boot']


class MainHandler(web.RequestHandler):
    def get(self):
        self.render("index.html", image_captions=image_captions)


class DetailHandler(web.RequestHandler):
    def get(self):
        image = self.get_argument('image', None)
        if not image:
            self.set_status(400)
            return self.finish("400: Missing image parameter")
        if image not in image_captions:
            self.set_status(404)
            return self.finish("404: Image not found")
        self.render("detail-snippet.html", image=image,
                    predictions=image_captions[image])


class CleanupHandler(web.RequestHandler):
    def get(self):
        self.render("cleanup.html")

    def delete(self):
        clean_up()


class UploadHandler(web.RequestHandler):
    def post(self):
        finish_ret = []
        new_files = self.request.files['file']
        for file_des in new_files:
            file_name = temp_img_prefix + file_des['filename']
            if valid_file_ext(file_name):
                rel_path = static_img_path + file_name
                output_file = open(rel_path, 'wb')
                output_file.write(file_des['body'])
                output_file.close()
                caption = run_ml(rel_path)
                finish_ret.append({
                    "file_name": rel_path,
                    "caption": caption[0]['caption']
                })
        if not finish_ret:
            self.send_error(400)
            return
        sort_image_captions()
        self.finish(json.dumps(finish_ret))


def valid_file_ext(filename):
    return '.' in filename and filename.split('.', 1)[1].lower() in VALID_EXT


# Runs ML on given image
def run_ml(img_path):
    img_file = {'image': open(img_path, 'rb')}
    image = preprocess_object(img_path)
    # Run curl command to send json to seldon
    features = [str(i+1) for i in range(0,784)]
    req = {"data": {"names": features, "ndarray": image}}
    results = requests.post(ml_endpoint, json=req)
    request_results = results.json()
    # Run postprocessing and retrieve results from returned json
    return_json = request_results['data']['ndarray']
    results = postprocess(return_json)
    cap_json = postpostprocess(results)
    print(cap_json)
    caption = cap_json['predictions']
    image_captions[img_path] = caption
    return caption

# preprocess and post processing images
def preprocess_object(image_path, target_shape=(28,28)):
    """

    image_path:    File path to image
    target_shape:  Shape that the model is expecting images to be in
                   by default the expected shape is (28,28)
    returns:       An array containing the raw data from the image at
                   'image_path' resized to 'target_shape'
    """

    image = read_image(image_path, target_shape)

    # Changes the image from a 1d array of size 784 to 2d of 28x28
    feature_list = []
    row = []
    for x in range(0,28):
        range1 = 28 * x
        row = []
        for vals in range(0,28):
            pixel = image[range1 + vals]
            row.append([pixel])
        feature_list.append(row)

    return [feature_list]

def read_image(image_path, target_shape):
    """

    image_path:    File path to image
    target_shape:  Shape that the model is expecting images to be in
                   by default the expected shape is (28,28)
    returns:       Returns a list of pixel values stored in 'L' mode
                   (black and white 8bit pixels)
    """
    image = Image.open(image_path)
    image = image.resize(target_shape)
    # Force image to be stored in 'L' mode (Black and White 8bit pixels)
    image = image.convert('L')
    raw = list(image.getdata())
    return raw

def postprocess(results):
    """

    results:    List of results where each result is a list of confidences
                in the image being predicted being of a certain class
    returns:    A list of 2-tuples where each 2-tuple corresponds to a result
                and is a 2-tuple where the second element is a list of sorted
                confidence values (from max to min) and the first element is
                the argsorted list of classes corresponding to the confidences
    """

    post_results = []
    for result in results:
        argsort_rev = numpy.argsort(result)[::-1]
        result_rev_sort = sorted(result)[::-1]
        post_results.append((argsort_rev, result_rev_sort))
    return post_results

def postpostprocess(post_results):
    """

    post_results:   An array of confidences that the image is of the class
                    with the same key as the index of the confidence value
    """

    FASHION_LIST = ['T-shirt/top','Trouser','Pullover','Dress','Coat','Sandal','Shirt','Sneaker','Bag','Ankle boot']

    cap_array = []
    for i in range(0,3):
        cap_array.append({"index": i, "caption": FASHION_LIST[post_results[0][0][i]],"probability": str(round(post_results[0][1][i]*100, 4))})
    cap_json = {'predictions': cap_array}
    return cap_json


def sort_image_captions():
    global image_captions
    image_captions = collections.OrderedDict(
        sorted(image_captions.items(), key=lambda t: t[0].lower()))


# Gets list of images with relative paths from static dir
def get_image_list():
    image_list = sorted(os.listdir(static_img_path))
    rel_img_list = [static_img_path + s for s in image_list]
    return rel_img_list


# Run all static images through ML
def prepare_metadata():
    threads = []

    rel_img_list = get_image_list()
    for img in rel_img_list:
        t = threading.Thread(target=run_ml, args=(img,))
        threads.append(t)

    for t in threads:
        t.start()

    for t in threads:
        t.join()

    sort_image_captions()


# Deletes all files uploaded through the GUI and removes them from the dict
def clean_up():
    img_list = get_image_list()
    for img_file in img_list:
        if img_file.startswith(static_img_path + temp_img_prefix):
            os.remove(img_file)
            image_captions.pop(img_file)


def signal_handler(sig, frame):
    ioloop.IOLoop.current().add_callback_from_signal(shutdown)


def shutdown():
    logging.info("Cleaning up image files")
    clean_up()
    logging.info("Stopping web server")
    server.stop()
    ioloop.IOLoop.current().stop()


def make_app():
    handlers = [
        (r"/", MainHandler),
        (r"/upload", UploadHandler),
        (r"/cleanup", CleanupHandler),
        (r"/detail", DetailHandler)
    ]

    configs = {
        'static_path': 'static',
        'template_path': 'templates'
    }

    return web.Application(handlers, **configs)


def main():
    parse_command_line()

    global ml_endpoint
    ml_endpoint = options.ml_endpoint
    # if '/model/predict' not in options.ml_endpoint:
    #     ml_endpoint = options.ml_endpoint + "/model/predict"

    logging.info("Connecting to ML endpoint at %s", ml_endpoint)

    # try:
    #     requests.get(ml_endpoint)
    # except requests.exceptions.ConnectionError:
    #     logging.error(
    #         "Cannot connect to the Image Caption Generator REST endpoint at " +
    #         options.ml_endpoint)
    #     raise SystemExit

    logging.info("Starting web server")
    app = make_app()
    global server
    server = httpserver.HTTPServer(app)
    server.listen(options.port)
    signal.signal(signal.SIGINT, signal_handler)

    logging.info("Preparing ML metadata (this may take some time)")
    start = time.time()
    prepare_metadata()
    end = time.time()
    logging.info("Metadata prepared in %s seconds", end - start)

    logging.info("Use Ctrl+C to stop web server")
    ioloop.IOLoop.current().start()


if __name__ == "__main__":
    main()
