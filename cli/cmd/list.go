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
)


// ListCmd is the struct to get all models.
type ListCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewListCmd is used to get all models.
func NewListCmd(ui terminal.UI, context plugin.PluginContext) *ListCmd {
	return &ListCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the model-list CLI command.
func (cmd *ListCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()
	cmd.ui.Say("Getting all models ...")

	c, err := NewDlaaSClient()
	if err != nil {
		lflog().Debugf("NewDlaaSClient failed: %s", err.Error())
		cmd.ui.Failed(err.Error())
	}
	lflog().Debugf("Calling ListModels")

	params := models.NewListModelsParams().WithTimeout(defaultOpTimeout)

	modelz, err := c.Models.ListModels(params, BasicAuth())

	if err != nil {
		lflog().WithError(err).Debugf("ListModels failed")
		var s string
		switch err.(type) {
		case *models.ListModelsUnauthorized:
			s = badUsernameOrPWD
		}
		responseError(s, err, cmd.ui)
		return nil
	}
	lflog().Debugf("Constructing table")
	table := cmd.ui.Table([]string{"ID", "Name", "Framework", "Training status", "Submitted", "Completed"})
	for _, v := range modelz.Payload.Models {
		ts := v.Training.TrainingStatus
		table.Add(v.ModelID, v.Name, v.Framework.Name + ":" + v.Framework.Version, ts.Status, formatTimestamp(ts.Submitted), formatTimestamp(ts.Completed))
	}
	table.Print()
	cmd.ui.Say("\n%d records found.", len(modelz.Payload.Models))
	return nil
}
