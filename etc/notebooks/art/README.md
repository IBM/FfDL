# Jupyter Notebook Using ART to Test the Robustness of Deep Learning Models

This [Jupyter](http://jupyter.org/install) notebook shows how to use the [Adversarial Robustness Toolbox (ART)](https://github.com/IBM/adversarial-robustness-toolbox)
to test the robustness of Deep Learning models against adversarial attacks.

## Prerequisites

To run this [notebook](ART_model_robustness_check.ipynb) you need a Kubernetes cluster with FfDL deployed as described
in the [FfDL/README.md](/README.md).

To store model and training data, this notebook requires access to a Cloud Object Storage (COS) instance.
[BlueMix Cloud Object Storage](https://console.bluemix.net/catalog/services/cloud-object-storage) offers a free 
*lite plan*. 
Follow [these instructions](https://dataplatform.ibm.com/docs/content/analyze-data/ml_dlaas_object_store.html)
to create your COS instance and generate [service credentials](https://console.bluemix.net/docs/services/cloud-object-storage/iam/service-credentials.html#service-credentials)
with [HMAC keys](https://console.bluemix.net/docs/services/cloud-object-storage/hmac/credentials.html#using-hmac-credentials).
Then go to the COS dashboard:
- Get the `cos_service_endpoint` from the **Endpoint** tab
- In the **Service credentials** tab, click **New Credential +** 
  - Add the "[HMAC](https://console.bluemix.net/docs/services/cloud-object-storage/hmac/credentials.html#using-hmac-credentials)"
    **inline configuration parameter**: `{"HMAC":true}`, click **Add**
  - Get the `access_key_id` (*AWS_ACCESS_KEY_ID*) and `secret_access_key` (*AWS_SECRET_ACCESS_KEY*) 
    from the `cos_hmac_keys` section of the instance credentials:
    ```
      "cos_hmac_keys": {
          "access_key_id": "1234567890abcdefghijklmnopqrtsuv",
          "secret_access_key": "0987654321zxywvutsrqponmlkjihgfedcba1234567890ab"
       }
    ```

## Setup

Before running this notebook for the first time we recommend creating a Python 3 *virtual environment* using either
[virtualenv](https://pypi.org/project/virtualenv/), [venv](https://docs.python.org/3/library/venv.html) (since Python 3.3),
or [Conda](https://conda.io/docs/user-guide/tasks/manage-environments.html).

```bash
# assuming present working directory to be the project root
pip3 install virtualenv
virtualenv .venv/art
.venv/art/bin/pip install -r etc/notebooks/art/requirements.txt --upgrade
```

## Running the Notebook

Before starting the Jupyter notebook server, make sure to activate the Python virtual environment.

```bash
source .venv/art/bin/activate
```

Start the Jupyter notebook server.

```bash
jupyter-notebook --notebook-dir etc/notebooks/art
# ... use Control-C to stop the notebook server
```

Deactivate the virtual environment after stopping the Jupyter notebook server.

```bash
deactivate
```

To delete the Python virtual environment run the following command:

```bash
rm -rf .venv/art
```

## Acknowledgements

Special thanks to [Anupama-Murthi](https://github.ibm.com/Anupama-Murthi) and [Vijay Arya](https://github.ibm.com/vijay-arya)
who created the original notebook which we modified here to showcase how to use *ART* with *FfDL*.
If you would like to try *[Watson Machine Learning (WML) Service](https://console.bluemix.net/catalog/services/machine-learning)* 
with *ART* check out Anupama and Vijay's notebook here:

[https://github.ibm.com/robust-dlaas/ART-in-WML/Use ART to check robustness of deep learning models.ipynb](https://github.ibm.com/robust-dlaas/ART-in-WML/blob/master/Use%20ART%20to%20check%20robustness%20of%20deep%20learning%20models.ipynb)
