#!/usr/bin/env bash

#!/bin/bash
operating_system=$(uname)
if [[ "$operating_system" == 'Linux' ]]; then
    CMD_SED=sed
elif [[ "$operating_system" == 'Darwin' ]]; then
    CMD_SED=gsed
fi
replacement_string=$(echo "$AWS_URL" | $CMD_SED -r 's/\//\\\//g')

echo $replacement_string
