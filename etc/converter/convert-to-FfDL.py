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

import json
from ruamel.yaml import YAML
import sys
import getopt

#global variables
language = ''
inputfile = ''
outputfile = ''
samplefile = ''
yaml=YAML(typ='safe')

def getFfDL():
	try:
		f = open(inputfile, "r")
		response = f.read()
		data = yaml.load(response)
		f.close()
		return data
	except:
		print("Missing {}".format(inputfile))

def getSampleJob():
	try:
		if samplefile:
			f = open(samplefile,"r")
		else:
			f = open("sample-FfDL.yaml","r")
		response = f.read()
		resYaml = yaml.load(response)
		f.close()
		return resYaml
	except:
		print("Missing sample-FfDL.yaml")

def createJob(sample,data):
	try:
		sample['framework']['name'] = data['model_definition']['framework']['name']
		sample['name'] = data['model_definition']['name']
		sample['description'] = data['model_definition']['description']
		sample['framework']['command'] = data['model_definition']['execution']['command']
		sample['data_stores'][0]['id'] = data['training_data_reference']['name']
		sample['data_stores'][0]['connection']['auth_url'] = data['training_data_reference']['connection']['endpoint_url']
		sample['data_stores'][0]['connection']['user_name'] = data['training_data_reference']['connection']['access_key_id']
		sample['data_stores'][0]['connection']['password'] = data['training_data_reference']['connection']['secret_access_key']
		sample['data_stores'][0]['training_data']['container'] = data['training_data_reference']['source']['bucket']
		sample['data_stores'][0]['training_results']['container'] = data['training_results_reference']['target']['bucket']
		py2 = False
		CPU = False
		try:
			if data['model_definition']['framework']['name'] == 'tensorflow':
				if '2.' in data['model_definition']['framework']['runtimes']['version']:
					py2 = True
		except:
			py2 = False

		try:
			sample['learners'] = int(data['model_definition']['execution']['compute_configuration']['nodes'])
		except:
			sample['learners'] =  1

		# Detect T-shirt requirements
		if data['model_definition']['execution']['compute_configuration']['name'] == "k80":
			sample['cpus'] = 4
			sample['gpus'] = 1
			sample['memory'] = '24Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "p100":
			sample['cpus'] = 8
			sample['gpus'] = 1
			sample['memory'] = '24Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "v100":
			sample['cpus'] = 26
			sample['gpus'] = 1
			sample['memory'] = '24Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "k80x2":
			sample['cpus'] = 8
			sample['gpus'] = 2
			sample['memory'] = '48Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "p100x2":
			sample['cpus'] = 16
			sample['gpus'] = 2
			sample['memory'] = '48Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "v100x2":
			sample['cpus'] = 52
			sample['gpus'] = 2
			sample['memory'] = '48Gb'
		elif data['model_definition']['execution']['compute_configuration']['name'] == "k80x4":
			sample['cpus'] = 16
			sample['gpus'] = 4
			sample['memory'] = '96Gb'
		else:
			CPU = True
			sample['cpus'] = 1
			sample['gpus'] = 0
			sample['memory'] = '1Gb'

		# Detect Framework version
		try:
			if data['model_definition']['framework']['name'] == 'tensorflow':
				if '1.3' in data['model_definition']['framework']['version']:
					if py2:
						if CPU:
							sample['framework']['version'] = "1.3.0"
						else:
							sample['framework']['version'] = "1.3.0-gpu"
					else:
						if CPU:
							sample['framework']['version'] = "1.3.0-py3"
						else:
							sample['framework']['version'] = "1.3.0-gpu-py3"
				elif '1.4' in data['model_definition']['framework']['version']:
					if py2:
						if CPU:
							sample['framework']['version'] = "1.4.0"
						else:
							sample['framework']['version'] = "1.4.0-gpu"
					else:
						if CPU:
							sample['framework']['version'] = "1.4.0-py3"
						else:
							sample['framework']['version'] = "1.4.0-gpu-py3"
				elif '1.5' in data['model_definition']['framework']['version']:
					if py2:
						if CPU:
							sample['framework']['version'] = "1.5.0"
						else:
							sample['framework']['version'] = "1.5.0-gpu"
					else:
						if CPU:
							sample['framework']['version'] = "1.5.0-py3"
						else:
							sample['framework']['version'] = "1.5.0-gpu-py3"
				else:
					if py2:
						if CPU:
							sample['framework']['version'] = "latest"
						else:
							sample['framework']['version'] = "latest-gpu"
					else:
						if CPU:
							sample['framework']['version'] = "latest-py3"
						else:
							sample['framework']['version'] = "latest-gpu-py3"
			elif data['model_definition']['framework']['name'] == 'caffe':
				if CPU:
					sample['framework']['version'] = "cpu"
				else:
					sample['framework']['version'] = "gpu"
			elif data['model_definition']['framework']['name'] == 'pytorch':
				sample['framework']['version'] = "latest"
		except:
			print("Wrong framework.version contents in {}".format(inputfile))

		if data['model_definition']['framework']['name'] != "tensorflow":
			sample.pop('evaluation_metrics', None)
	except:
		print("Missing contents in {}".format(inputfile))
	try:
		if outputfile:
			f = open(outputfile, "w")
		else:
			f = open("manifest-FfDL.yaml", "w")
		yaml.default_flow_style = False
		yaml.dump(sample, f)
		f.close()
	except:
		if outputfile:
			print("Cannot write contents to {}".format(outputfile))
		else:
			print("Cannot write contents to manifest-FfDL.yaml.")


if __name__ == "__main__":
	argv = sys.argv[1:]
	try:
	  opts, args = getopt.getopt(argv,"i:o:s:",["ifile=","ofile=","sfile="])
	except getopt.GetoptError:
	  print('Format Error: Wrong format.')
	  print('convert-to-FfDL.py -i <inputfile> -o <outputfile> -s <samplefile>')
	  sys.exit(2)
	for opt, arg in opts:
		if opt in ("-i", "--ifile"):
		 inputfile = arg
		elif opt in ("-o", "--ofile"):
		 outputfile = arg
		elif opt in ("-s", "--sfile"):
		 samplefile = arg
	if not inputfile:
	  print('Input Error: inputfile cannot be empty.')
	  print('convert-to-FfDL.py -i <inputfile> -o <outputfile> -s <samplefile>')
	  sys.exit(2)
	data = getFfDL()
	sample = getSampleJob()
	createJob(sample,data)
