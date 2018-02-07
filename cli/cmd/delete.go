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
	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client/models"
)

// DeleteCmd is the struct to delete a model.
type DeleteCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewDeleteCmd is used to delete a model.
func NewDeleteCmd(ui terminal.UI, context plugin.PluginContext) *DeleteCmd {
	return &DeleteCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the model-delete CLI command.
func (cmd *DeleteCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Argument MODEL_ID missing")
	} else {
		modelID := args[0]

		cmd.ui.Say("Deleting model '%s'...", terminal.EntityNameColor(modelID))
		c, err := NewDlaaSClient()
		if err != nil {
			cmd.ui.Failed(err.Error())
		}
		params := models.NewDeleteModelParams().
			WithModelID(modelID).
			WithTimeout(defaultOpTimeout)

		_, err = c.Models.DeleteModel(params, BasicAuth())

		if err != nil {
			var s string
			switch err.(type) {
			case *models.DeleteModelUnauthorized:
				s = badUsernameOrPWD
			case *models.DeleteModelNotFound:
				s = "Model not found."
			}
			responseError(s, err, cmd.ui)
			return nil
		}
		cmd.ui.Ok()
	}
	return nil
}
