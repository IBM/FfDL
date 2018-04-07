<!--
{% comment %}
Copyright 2017-2018 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
{% endcomment %}
-->

A Docker image to transfer data to and from an S3 Object Storage.

In addition to the [common input variables](../README.md) defined in the parent directory, the Object Storage specific inputs are passed in the following environment variables.

- For the `load.sh` script: pointer to a an S3 Object Storage bucket to download:
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_BUCKET`:

- For the `store.sh` script: pointer to an S3 Object Storage bucket to upload to:
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_BUCKET`:

- This image has an additional script, called `loadmodel.sh`, that downloads a model definition. The model definition is assumed to be in a zip file stored in a single object. This file is unzipped after downloading.
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_OBJECT`:

Use files with `ppc64le` extensions to compile and run the tests on POWER machine. The tests assume that the `aws` and `swift` cli tools are available on the host.

