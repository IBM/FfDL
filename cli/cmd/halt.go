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
	"github.com/urfave/cli"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
	"github.com/IBM/FfDL/restapi/api_v1/client/models"
	"github.com/IBM/FfDL/restapi/api_v1/restmodels"
)

// HaltCmd is the struct to get a training job.
type HaltCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewHaltCmd is used to get a training job's info.
func NewHaltCmd(ui terminal.UI, context plugin.PluginContext) *HaltCmd {
	return &HaltCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the training-show CLI command.
func (cmd *HaltCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Argument MODEL_ID missing")
	} else {
		modelID := args[0]

		cmd.ui.Say("Halting training job '%s'...", terminal.EntityNameColor(modelID))
		c, err := NewDlaaSClient()
		if err != nil {
			cmd.ui.Failed(err.Error())
		}

		params := models.NewPatchModelParamsWithTimeout(defaultOpTimeout).
				WithModelID(modelID).
				WithPayload(&restmodels.TrainingUpdate{
					Status: "halt",
				})
		_, err = c.Models.PatchModel(params, basicAuth)

		if err != nil {
			var s string
			switch err.(type) {
			case *models.PatchModelUnauthorized:
				s = "Bad username or password."
			case *models.PatchModelNotFound:
				s = "Model ID not found."
			}
			responseError(s, err, cmd.ui)
		}
		cmd.ui.Ok()
	}
	return nil
}
