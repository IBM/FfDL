#!/bin/bash
# fail fast
set -e
export MAKE_ARGS=--no-print-directory
# compile and build Docker images
glide -q install
make $MAKE_ARGS build
make $MAKE_ARGS docker-build
# deploy services
make $MAKE_ARGS deploy
# submit a test job
make $MAKE_ARGS test-submit
