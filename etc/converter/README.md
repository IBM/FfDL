# Deploy FfDL Models to Watson Studio, and vice versa

Since Watson Studio Deep Learning and FfDL use different model definition file i.e. manifest.yml to define their training jobs, please use this simple script to help you convert between the two different version of the manifest.yml. The [convert-to-WML.py](convert-to-WML.py) and [convert-to-FfDL.py](convert-to-FfDL.py) are the conversion scripts for converting your FfDL training job's manifest.yml to Watson Studio Deep Learning format and vice versa.

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

  # Convert FfDL manifest.yml to Watson Studio Deep Learning format
	python convert-to-WML.py -i <inputfile> -o <outputfile> -s <samplefile>

  # Convert Watson Studio Deep Learning manifest.yml to FfDL format
  python convert-to-FfDL.py -i <inputfile> -o <outputfile> -s <samplefile>
	```
	Now, a converted <outputfile> file should be created with all the information in your original manifest file.

4. Copy the new YAML file and use it for your FfDL/Watson Studio Deep Learning training job.

5. Note that all the T-shirt size in Watson Studio Deep Learning requires GPU, so that will be the default conversion. If you only want to run on CPU, please modify the `gpus` section to 0 along with `cpus` and `memory` based on your need. Also change the framework version with the one enabled in CPU. You can find the list of CPU framework version at [user-guide.md](../../docs/user-guide.md#1-supported-deep-learning-frameworks). Below is the T-shirt size table between Watson Studio Deep Learning and FfDL.

| T-shirt Tiers     | GPUs    | RAM (GB) | CPUs |
| ------------- | ------------- | --------------- | --------------- |
| k80 | 1 | 24 | 4 |
| k80x2 | 2 | 48 | 8 |
| k80x4 | 4 | 96 | 16 |
| p100 | 1 | 24 | 8 |
| p100x2 | 2 | 48 | 16 |
| v100 | 1 | 24 | 26 |
| v100x2 | 2 | 48 | 52 |

## Troubleshooting

- If you are converting Watson Studio Deep Learning yml to FfDL format, both `training_data_reference` and `training_results_reference` need to be in the same object storage (could be different bucket) because FfDL only takes one object storage connection.

- In Watson Studio Deep Learning, TensorFlow version is only available up to 1.5

- Caffe2 is not available yet in Watson Studio Deep Learning. Thus, the conversion script won't take any caffe2 input.

- The conversion script won't take `small`, `medium`, and `large` T-shirt size because they will be deprecated soon.

## Example Manifest.yml

The example FfDL manifest.yml is the [sample-FfDL.yaml](sample-FfDL.yaml). The description for each field is available at the [user-guide.md](../../docs/user-guide.md#24-creating-manifest-file).

The example Watson Machine Learning manifest.yml is the [sample-WML.yaml](sample-WML.yaml). The description for each field is available at the [model definition guide](https://dataplatform.ibm.com/docs/content/analyze-data/ml_dlaas_working_with_training_run.html?audience=wdp&linkInPage=true) at Watson Data Platform.
