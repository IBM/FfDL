#!/bin/bash
# fail fast
set -e
export MAKE_ARGS=--no-print-directory
# Open SSH
#  - echo travis:$sshpassword | sudo chpasswd
#  - sudo sed -i 's/ChallengeResponseAuthentication no/ChallengeResponseAuthentication yes/' /etc/ssh/sshd_config
#  - sudo service ssh restart
#  - sudo apt-get install sshpass
#  - sshpass -p $sshpassword ssh -R 9999:localhost:22 -o StrictHostKeyChecking=no travis@$bouncehostip
# compile and build Docker images
glide -q install
make $MAKE_ARGS docker-build-base
make $MAKE_ARGS gen-certs
make $MAKE_ARGS build
make $MAKE_ARGS docker-build
# deploy services
make $MAKE_ARGS deploy
# submit a test job
make $MAKE_ARGS test-submit-minikube-ci
