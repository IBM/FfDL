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
	"fmt"
	"os"
	"strings"

	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client/models"

	"github.com/urfave/cli"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
	"path/filepath"
)

// TrainCmd is the struct to deploy a model.
type TrainCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewTrainCmd is used to deploy a model's.
func NewTrainCmd(ui terminal.UI, context plugin.PluginContext) *TrainCmd {
	return &TrainCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the model-deploy CLI command.
func (cmd *TrainCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()
	if len(args) != 2 {
		cmd.ui.Failed("Incorrect number of arguments.")
	}
	params := models.NewPostModelParams().WithTimeout(defaultOpTimeout)

	params.WithManifest(openManifestFile(cmd.ui, args[0]))

	_, manifestFile := filepath.Split(args[0])

	if strings.Contains(args[1], ".zip") {
		modelDefinitionFile := args[1]
		cmd.ui.Say("Deploying model with manifest '%s' and model file '%s'...", terminal.EntityNameColor(manifestFile), terminal.EntityNameColor(modelDefinitionFile))
		params.WithModelDefinition(openModelDefinitionFile(cmd.ui, modelDefinitionFile))
	} else {

		modelDir := args[1]
		cmd.ui.Say("Deploying model with manifest '%s' and model files in '%s'...", terminal.EntityNameColor(manifestFile), terminal.EntityNameColor(modelDir))
		f, err2 := zipit(modelDir + "/")
		if err2 != nil {
			cmd.ui.Failed("Unexpected error when compressing model directory: %v", err2)
			os.Exit(1)
		}

		// reopen the file (I tried to leave the file open but it did not work)
		zip, err := os.Open(f.Name())
		if err != nil {
			fmt.Println("Error open temporary ZIP file: ", err)
		}
		defer os.Remove(zip.Name())

		params.WithModelDefinition(zip)
	}

	c, err := NewDlaaSClient()
	if err != nil {
		cmd.ui.Failed(err.Error())
	}

	response, err := c.Models.PostModel(params, BasicAuth())

	if err != nil {
		var s string
		switch err.(type) {
		case *models.PostModelUnauthorized:
			s = badUsernameOrPWD
		case *models.PostModelBadRequest:
			if resp, ok := err.(*models.PostModelBadRequest); ok {
				if resp.Payload != nil { // we may not have a payload
					s = fmt.Sprintf("Error: %s. %s", resp.Payload.Error, resp.Payload.Description)
				} else {
					s = fmt.Sprintf("Bad request: %s", err.Error())
				}
			}
		}
		responseError(s, err, cmd.ui)
		return nil
	}

	id := LocationToID(response.Location)
	cmd.ui.Say("Model ID: %s", terminal.EntityNameColor(id))
	cmd.ui.Ok()

	return nil
}
