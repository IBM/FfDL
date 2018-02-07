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

