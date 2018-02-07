/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/urfave/cli"

	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
)

const (
	manifestFile = "manifest.yml"
)

// InitCmd is the struct to create a new project template.
type InitCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewInitCmd is used to create a new project template.
func NewInitCmd(ui terminal.UI, context plugin.PluginContext) *InitCmd {
	return &InitCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the project-template CLI command.
func (cmd *InitCmd) Run(ctx *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := ctx.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Not enough arguments.")
	} else {

		projectName := args[0]

		if _, err := os.Stat(projectName); err == nil {
			cmd.ui.Failed("Project directory '%s' already exists", terminal.EntityNameColor(projectName))
		}

		cmd.ui.Say("Creating project '%s'...", terminal.EntityNameColor(projectName))

		if err := os.Mkdir(projectName, 0755); err != nil {
			cmd.ui.Failed("Error creating directory '%s'", terminal.EntityNameColor(projectName))
		}

		data := make(map[string]string)
		data["name"] = projectName
		data["version"] = "1.0.0"
		data["description"] = "Your first deep learning model"
		data["gpus"] = "1"
		data["memory"] = "2GB"
		data["fwname"] = stringOrDefault(ctx.String("type"), "tensorflow")
		data["fwversion"] = stringOrDefault(ctx.String("version"), "1.0")

		cmd.ui.Say("  adding: %s", manifestFile)
		if err := cmd.writeManifest(projectName, data); err != nil {
			cmd.ui.Failed("Error creating '%s'", terminal.EntityNameColor("manifest.yml"))
		}

		cmd.ui.Ok()
	}
	return nil
}

func (cmd *InitCmd) writeManifest(dir string, data map[string]string) error {
	filename := filepath.Join(dir, manifestFile)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	t, err := template.New("manifest").Parse(manifest)
	if err != nil {
		return err
	}

	return t.Execute(file, data)
}

const manifest = `
# Define the model metadata.
model_definition:
  name: {{.name}}
  description: My first model

# Optional location parameter with a reference to a data store ID that contains
# the model definition. If provided, the model definition will be loaded from
# the data store. Otherwise,the model definition has to be submitted via the
# POST /models request.
#location: <reference to data store ID>

framework:
	name: TBD
	version: "TBD"

# Defines the training parameters
training:
# The command to execute during training. $DATA_DIR is guaranteed to contain
# all the input data when the training starts. This may include training data
# and pre-trained models
	command: >
		python3 convolutional_network.py --trainImagesFile ${DATA_DIR}/train-images-idx3-ubyte.gz
		--trainLabelsFile ${DATA_DIR}/train-labels-idx1-ubyte.gz
		--testImagesFile ${DATA_DIR}/t10k-images-idx3-ubyte.gz
		--testLabelsFile ${DATA_DIR}/t10k-labels-idx1-ubyte.gz
		--learningRate 0.001
		--trainingIters 6000

# Resource requirements can either be specified by providing detailed requirements
# on training such as gpus, cpus, mem and the number of learners.
gpus: 2
memory: 2000MiB
learners: 1

# Alternative to gpu, cpu, memory, learners is to specify a "training deployment" size
# small —> 1 gpu, 1 learner, medium—> 2 gpus, 1 learner, large —> 4 gpu, 1 learner,
# 2xlarge—> 4 gpu, 2 learners, 4xlarge —> 4 gpu, 4 learners.
size: medium

# Input data for the training. Values have to be valid storage references.
# The input can contain training data as well as pre-trained models.
input_data:
- mnist_data_store
- mnist_pretrained_store
# Output data from the training. Values have to be valid storage references.
# Currently we only support one data store but this might evolve in the future.
output_data:
- mnist_training_store # enforced to be one for now

# Data stores for training data, store/retrieve models and model definitions.
# Currently, we only support the IBM Object Store flavors.
data_stores:
	- data_store_id: mnist_data_store
		type: softlayer_objectstore
		bucket: mnist_lmdb_data
		# Optional prefix to limit what gets downloaded from a bucket
		# prefix: /version12
		connection:
			auth_url: https://dal05.objectstorage.service.networklayer.com/auth/v1.0/
			user_name: <PLEASE_INSERT>
			password: <PLEASE_INSERT>

	- data_store_id: mnist_pretrained_store
		type: softlayer_objectstore
		bucket: mnist_training_results
		connection:
			$ref: mnist_data_store.connection # allows to re-use connection info from another data_store.

	- data_store_id: mnist_data_store
		type: softlayer_objectstore
		bucket: mnist_pretrained_models
		connection:
			$ref: mnist_data_store.connection
`
