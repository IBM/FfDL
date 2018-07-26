#!/bin/bash
# fail fast
set -e
export MAKE_ARGS=--no-print-directory
# get pre-built images
make $MAKE_ARGS pull-prebuilt-images
export IMAGE_TAG=$TRAVIS_IMAGE_VERSION
# deploy services
make $MAKE_ARGS deploy
# submit a test job
make $MAKE_ARGS test-submit-minikube-ci
