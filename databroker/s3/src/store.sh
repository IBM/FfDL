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


# Upload files from $DATA_DIR to Object Storage.

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$SCRIPTDIR/utility.sh"

trap panic ERR # exit immediately on error

# Validate input.
: "${DATA_DIR:?Need to set DATA_DIR to non-empty value}"
: "${DATA_STORE_BUCKET:?Need to set DATA_STORE_BUCKET to non-empty value}"
: "${DATA_STORE_USERNAME:?Need to set DATA_STORE_USERNAME to non-empty value}"
: "${DATA_STORE_PASSWORD:?Need to set DATA_STORE_PASSWORD to non-empty value}"
: "${DATA_STORE_AUTHURL:?Need to set DATA_STORE_AUTHURL to non-empty value}"

# For S3 Object Storage
export AWS_ACCESS_KEY_ID=$DATA_STORE_USERNAME
export AWS_SECRET_ACCESS_KEY=$DATA_STORE_PASSWORD

env |sort

echo Using Object Storage account $DATA_STORE_USERNAME at $DATA_STORE_AUTHURL

# Upload data.
echo Upload start: $(date)
echo "Uploading from $DATA_DIR to bucket $DATA_STORE_BUCKET"
mkdir -p "$DATA_DIR"
time with_backoff aws --endpoint-url=$DATA_STORE_AUTHURL s3 sync "$DATA_DIR" "s3://$DATA_STORE_BUCKET"
echo Upload end: $(date)
