# fail fast
set -e
export MAKE_ARGS=--no-print-directory
# pull images from dockerhub
make $MAKE_ARGS pull-dockerhub-images

# deploy services
make $MAKE_ARGS deploy
# submit a test job
make $MAKE_ARGS test-submit
