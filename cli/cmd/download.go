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
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/urfave/cli"
	"github.com/IBM/FfDL/restapi/api_v1/client/models"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
)

// DownloadCmd is the struct to download a trained model
type DownloadCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewDownloadCmd is used to get a training job's info.
func NewDownloadCmd(ui terminal.UI, context plugin.PluginContext) *DownloadCmd {
	return &DownloadCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the training-show CLI command.
func (cmd *DownloadCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()
	modelDefinition := cliContext.Bool("definition")
	trainedModel := cliContext.Bool("trainedmodel")

	if len(args) == 0 || cliContext.NumFlags() == 0 {
		cmd.ui.Failed("Incorrect arguments")
		return nil
	} else if modelDefinition && trainedModel {
		cmd.ui.Failed("Incorrect arguments: Specify either --definition or --trainedmodel.")
		return nil
	}

	trainingID := args[0]
	filename := cliContext.String("filename")
	c, err := NewDlaaSClient()
	if err != nil {
		cmd.ui.Failed(err.Error())
	}
	payload := bytes.NewBuffer(nil)


	if trainedModel {

		cmd.ui.Say("Getting trained model for '%s'...", terminal.EntityNameColor(trainingID))

		// set high timeout because of file download
		params := models.NewDownloadTrainedModelParamsWithTimeout(time.Hour).
				WithModelID(trainingID)
		_, err = c.Models.DownloadTrainedModel(params, BasicAuth(), payload)

		if err != nil {
			var s string
			switch err.(type) {
			case *models.DownloadTrainedModelUnauthorized:
				s = badUsernameOrPWD
			case *models.DownloadTrainedModelNotFound:
				s = "Trained model not found."
			}
			responseError(s, err, cmd.ui)
			return nil
		}

		if filename == "" {
			filename = fmt.Sprintf("%s-trainedmodel.zip", trainingID)
		}

	} else if modelDefinition {

		cmd.ui.Say("Getting model definition for '%s'...", terminal.EntityNameColor(trainingID))

		// set high timeout because of file download
		params := models.NewDownloadModelDefinitionParams().
				WithModelID(trainingID).
				WithTimeout(time.Hour)
		_, err = c.Models.DownloadModelDefinition(params, BasicAuth(), payload)

		if err != nil {
			var s string
			switch err.(type) {
			case *models.DownloadModelDefinitionUnauthorized:
				s = badUsernameOrPWD
			case *models.DownloadModelDefinitionNotFound:
				s = "Model definition not found."
			}
			responseError(s, err, cmd.ui)
			return nil
		}

		if filename == "" {
			filename = fmt.Sprintf("%s-definition.zip", trainingID)
		}
	}

	if err := ioutil.WriteFile(filename, payload.Bytes(), 0644); err != nil {
		cmd.ui.Failed("Error getting trained model: %v", err)
		return nil
	}

	cmd.ui.Say("Downloaded file: %s", terminal.EntityNameColor(filename))
	cmd.ui.Ok()
	return nil
}
