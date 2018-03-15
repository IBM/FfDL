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
			f = open("sample-WML.yaml","r")
		response = f.read()
		resYaml = yaml.load(response)
		f.close()
		return resYaml
	except:
		print("Missing sample-WML.yaml")

def createJob(sample,data):
	try:
		sample['model_definition']['framework']['name'] = data['framework']['name']
		sample['model_definition']['name'] = data['name']
		sample['model_definition']['description'] = data['description']
		sample['model_definition']['execution']['command'] = data['framework']['command']
		sample['training_data_reference']['name'] = data['data_stores'][0]['id']
		sample['training_data_reference']['connection']['endpoint_url'] = data['data_stores'][0]['connection']['auth_url']
		sample['training_data_reference']['connection']['access_key_id'] = data['data_stores'][0]['connection']['user_name']
		sample['training_data_reference']['connection']['secret_access_key'] = data['data_stores'][0]['connection']['password']
		sample['training_data_reference']['source']['bucket'] = data['data_stores'][0]['training_data']['container']
		sample['training_results_reference']['name'] = data['data_stores'][0]['id']
		sample['training_results_reference']['connection']['endpoint_url'] = data['data_stores'][0]['connection']['auth_url']
		sample['training_results_reference']['connection']['access_key_id'] = data['data_stores'][0]['connection']['user_name']
		sample['training_results_reference']['connection']['secret_access_key'] = data['data_stores'][0]['connection']['password']
		sample['training_results_reference']['target']['bucket'] = data['data_stores'][0]['training_results']['container']
		if data['framework']['name'] == 'tensorflow':
			if 'py3' in data['framework']['version']:
				sample['model_definition']['framework']['runtimes']['version'] = "3.5"
			else:
				sample['model_definition']['framework']['runtimes']['version'] = "2.7"
		else:
			sample['model_definition']['framework']['runtimes']['version'] = "3.5"
		sample['model_definition']['execution']['compute_configuration']['nodes'] = int(data['learners']) if data.get("learners") else 1
		gpus = int(data['gpus']) if data.get("gpus") else 0
		if gpus <= 1:
			if int(data['cpus']) <= 4:
				sample['model_definition']['execution']['compute_configuration']['name'] = "k80"
			elif int(data['cpus']) <= 8:
				sample['model_definition']['execution']['compute_configuration']['name'] = "p100"
			else:
				sample['model_definition']['execution']['compute_configuration']['name'] = "v100"
		elif gpus <= 2:
			if int(data['cpus']) <= 8:
				sample['model_definition']['execution']['compute_configuration']['name'] = "k80x2"
			elif int(data['cpus']) <= 16:
				sample['model_definition']['execution']['compute_configuration']['name'] = "p100x2"
			else:
				sample['model_definition']['execution']['compute_configuration']['name'] = "v100x2"
		else:
			sample['model_definition']['execution']['compute_configuration']['name'] = "k80x4"
	except:
		print("Missing contents in {}".format(inputfile))
	try:
		if data['framework']['name'] == 'tensorflow':
			if '1.3' in data['framework']['version']:
				sample['model_definition']['framework']['version'] = "1.3"
			elif '1.4' in data['framework']['version']:
				sample['model_definition']['framework']['version'] = "1.4"
			elif '1.5' in data['framework']['version']:
				sample['model_definition']['framework']['version'] = "1.5"
			else:
				sample['model_definition']['framework']['version'] = "1.5"
		elif data['framework']['name'] == 'caffe':
			sample['model_definition']['framework']['version'] = "1.0"
		elif data['framework']['name'] == 'pytorch':
			sample['model_definition']['framework']['version'] = "0.3"
	except:
		print("Wrong framework.version contents in {}".format(inputfile))
	try:
		if outputfile:
			f = open(outputfile, "w")
		else:
			f = open("manifest-WML.yaml", "w")
		yaml.default_flow_style = False
		yaml.dump(sample, f)
		f.close()
	except:
		if outputfile:
			print("Cannot write contents to {}".format(outputfile))
		else:
			print("Cannot write contents to manifest-WML.yaml.")


if __name__ == "__main__":
	argv = sys.argv[1:]
	try:
	  opts, args = getopt.getopt(argv,"i:o:s:",["ifile=","ofile=","sfile="])
	except getopt.GetoptError:
	  print('Format Error: Wrong format.')
	  print('convert-to-WML.py -i <inputfile> -o <outputfile> -s <samplefile>')
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
	  print('convert-to-WML.py -i <inputfile> -o <outputfile> -s <samplefile>')
	  sys.exit(2)
	data = getFfDL()
	sample = getSampleJob()
	createJob(sample,data)
