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

(Note: The implementation is lagging the spec in the README.)

A Docker image to transfer data to and from Softlayer Object Storage.

In addition to the [common input variables](../README.md) defined in the parent directory, the Object Storage specific inputs are passed in the following environment variables.

- For the `load.sh` script: pointer to a Softlayer Object Storage bucket to download:
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_BUCKET`:

- For the `store.sh` script: pointer to a Softlayer Object Storage bucket to upload to:
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_BUCKET`:

- This image has an additional script, called `loadmodel.sh`, that downloads a model definition. The model definition is assumed to be in a zip file stored in a single object. This file is unzipped after downloading.
  - `DATA_STORE_USERNAME`:
  - `DATA_STORE_PASSWORD`:
  - `DATA_STORE_AUTHURL`:
  - `DATA_STORE_OBJECT`:

To use a Bluemix Object Storage account, pass these additional variables to the above scripts:
  - `DATA_STORE_PROJECTID`
  - `DATA_STORE_DOMAINNAME`
  - `DATA_STORE_REGION`

Use files with `ppc64le` extenions to compile and run on POWER machines.


# Versions

Version   | Notes
--------- | -----
dev_v1    | Initial version.
dev_v2    | Remove special handling for log files.
dev_v2.1  | Add retry logic and more log output.
