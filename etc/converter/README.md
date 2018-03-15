# Overview

Since Watson Machine Learning and FfDL use different model definition manifest.yml to define their training jobs, we provided a simple script to help you convert between the two different version of the manifest.yml. The [convert-to-WML.py](convert-to-WML.py) and [convert-to-FfDL.py](convert-to-FfDL.py) are the conversion scripts for converting your FfDL training job's manifest.yml to Watson Machine Learning format and vice versa.

## Instructions

1. Clone and go into this directory
	```bash
	git clone https://github.com/IBM/FfDL
	cd FfDL/converter
	```

2. Use the following commands to install the necessary Python packages and run the Python Job to build your custom Watson Machine Learning/FfDL `manifest.yml`.

* ```<inputfile>:``` The manifest file you want to convert.
* ```<outputfile>:``` The filename for the converted manifest file. Default is `manifest-WML.yaml`/`manifest-FfDL.yaml`.
* ```<samplefile>:``` The sample manifest format file with all the default values. Default is `sample-WML.yaml`/`sample-FfDL.yaml`.

	```bash
	pip install -r requirement.txt

  # Convert FfDL manifest.yml to Watson Machine Learning format
	python convert-to-WML.py -i <inputfile> -o <outputfile> -s <samplefile>

  # Convert Watson Machine Learning manifest.yml to FfDL format
  python convert-to-FfDL.py -i <inputfile> -o <outputfile> -s <samplefile>
	```
	Now, a converted <outputfile> file should be created with all the information in your original manifest file.

4. Copy the new YAML file and use it for your FfDL/Watson Machine Learning training job.

## Troubleshooting

- For converting to FfDL format, both `training_data_reference` and `training_results_reference` need to be in the same object storage (Could be different bucket) because FfDL only takes one object storage connection.

- In Watson Machine Learning, TensorFlow version only available up to 1.5

## Example Manifest.yml

The example FfDL manifest.yml is the [sample-FfDL.yaml](sample-FfDL.yaml). The description for each field is available at the [user-guide.md](../../docs/user-guide.md).

The example Watson Machine Learning manifest.yml is the [sample-WML.yaml](sample-WML.yaml). The description for each field is available at the [model definition guide](https://dataplatform.ibm.com/docs/content/analyze-data/ml_dlaas_working_with_training_run.html?audience=wdp&linkInPage=true) at Watson Data Platform.
