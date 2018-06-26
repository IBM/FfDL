#!/usr/bin/env bash

staging=$GOPATH/src/github.com/IBM/FfDL

declare -a services=(
                    "${staging}/metrics/docker/certs" \
                    "${staging}/metrics/log_collectors/training_data_service_client/certs" \
                    "${staging}/restapi/certs" \
                    "${staging}/lcm/certs" \
                    "${staging}/jobmonitor/certs" \
                    "${staging}/trainer/docker/certs" \
                    )
for dir in ${services[@]}
do
    echo "------ ${dir}"
    mkdir -p ${dir}
    cp ca.crt ${dir}
    cp server.crt ${dir}
    cp server.key ${dir}
done
