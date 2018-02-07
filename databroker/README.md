This repository contains Docker images that are used to transfer data to and from an external store.

To download data from an external store, the Docker images have the following input and output:

- input: A set of environment variables:
  - `DATA_DIR`: The path in the container to store the downloaded files. This may be a mounted Docker volume.
  - `DATA_STORE_xxx`: A set of variables specific to the data broker that point to the external store and credentials to access the store.

- `load.sh`: An executable file in the image that handles the process of downloading the data from the external store to the `DATA_DIR`.
  - This is potentially a long running process.
  - The script must be idempotent since the container may be restarted with partially downloaded data already in the `DATA_DIR`.
  - The script should exit with 0 on success or non-zero on failure.
  - Any progress logs and error messages should be written to stdout.

- output of `load.sh`: The downloaded data should be in the `DATA_DIR` on success. The contents of the `DATA_DIR` are undefined on error.

Note that the intention is to use the `load.sh` script to download both training data and model code. This works with the new API where the model is also stored as separate objects in Object Storage, and hence doesn't require a unzip post-download step.

Similarly, to upload data to an external store, the Docker images have the following input and output:

- input: A set of environment variables:
  - `DATA_DIR`: The path in the container that contains the files to upload. This may be a mounted Docker volume.
  - `DATA_STORE_xxx`: A set of variables specific to the data broker that point to the external store and credentials to access the store.

- `store.sh`: An executable file in the image that handles the process of uploading the data from the `DATA_DIR` to the external store.
  - This is potentially a long running process.
  - The script must be idempotent since the container may be restarted with partially uploaded data already in the external data store.
  - The script should exit with 0 on success or non-zero on failure.
  - Any progress logs and error messages should be written to stdout.

- output of `store.sh`: The uploaded data should be in the external data store on success. The contents of the store are undefined on error.
