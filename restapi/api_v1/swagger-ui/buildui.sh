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


# Build a static Web site to serve the DLaaS Swagger.
# The site can be served locally, or deployed as a CloudFoundry app.

SCRIPTDIR="$(cd $(dirname "$0")/ && pwd)"

# Parse command line arguments
BUILDDIR=${1:-"$SCRIPTDIR"/build} # defaults to "build" if not specified


# Build the site

echo Building Swagger UI in directory: $BUILDDIR
SITEDIR="$BUILDDIR/dlaas-api"
mkdir -p "$BUILDDIR"
rm -rf "$SITEDIR"

id=$(docker create schickling/swagger-ui) # Get Swagger-ui files from container.
docker cp $id:/app/ "$SITEDIR"/
docker rm $id > /dev/null

cp -a "$SCRIPTDIR/manifest.yml" "$SITEDIR/"
cp -a "$SCRIPTDIR/../swagger/swagger.yml" "$SITEDIR/"
cat "$SITEDIR/index.html" |sed 's|http://petstore.swagger.io/v2/swagger.json|./swagger.yml|' > "$SITEDIR/index.html.$$"
mv "$SITEDIR/index.html.$$" "$SITEDIR/index.html"

# # Copy sample models
 for model in torch-mnist-model tf-model caffe-mnist-model caffe-inc-model keras-model mxnet-model; do
 	cp -a "$SCRIPTDIR/../../tests/testdata/$model" "$BUILDDIR/"
 	(cd "$BUILDDIR/$model" && zip -r ../$model.zip .)
 	cp -a "$BUILDDIR/$model.zip" "$SITEDIR/"
 done

# Print instructions

echo "To serve the site locally: (cd \"$SITEDIR\" && python -m SimpleHTTPServer)"

echo "To deploy as a Bluemix app: (cd \"$SITEDIR\" && cf api https://api.stage1.ng.bluemix.net && cf target -o dlaas -s dev && cf push)"
