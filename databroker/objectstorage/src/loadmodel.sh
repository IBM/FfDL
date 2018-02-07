#!/bin/bash

#
# Copyright 2017-2018 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


# Download model from Object Storage to $DATA_DIR.

# Validate input.
: "${DATA_DIR?Need to set DATA_DIR}"
: "${DATA_STORE_OBJECT:?Need to set DATA_STORE_OBJECT to non-empty value}"
: "${DATA_STORE_USERNAME:?Need to set DATA_STORE_USERNAME to non-empty value}"
: "${DATA_STORE_PASSWORD:?Need to set DATA_STORE_PASSWORD to non-empty value}"
: "${DATA_STORE_AUTHURL:?Need to set DATA_STORE_AUTHURL to non-empty value}"

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

trap panic ERR # exit immediately on error

constructSwiftConnectionArgs
echo Connection args: "${SWIFT_CONNECTION_ARGS[@]}"

echo Using Object Storage account $DATA_STORE_USERNAME at $DATA_STORE_AUTHURL

bucket=$(echo "$DATA_STORE_OBJECT" |cut -d / -f 1)
object=$(echo "$DATA_STORE_OBJECT" |cut -d / -f 2-)

# Download data.
echo Download start: $(date)
echo "Downloading object $DATA_STORE_OBJECT to $DATA_DIR"
time with_backoff swift --verbose "${SWIFT_CONNECTION_ARGS[@]}" download -o /tmp/model.zip "$bucket" "$object"
mkdir -p "$DATA_DIR"
cd "$DATA_DIR"
unzip /tmp/model.zip
echo Download end: $(date)
chmod -R 777 .
