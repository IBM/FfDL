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
	"strings"
	"encoding/json"
	"bytes"
	"os"
)

// ShowCmd is the struct to get details of the model/training.
type ShowCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewShowCmd is used to get a model's info.
func NewShowCmd(ui terminal.UI, context plugin.PluginContext) *ShowCmd {
	return &ShowCmd{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the model-show CLI command.
func (cmd *ShowCmd) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Argument MODEL_ID missing")
	} else {
		modelID := args[0]
		isJSON := cliContext.IsSet("json")

		cmd.ui.Say("Getting model '%s'...", terminal.EntityNameColor(modelID))
		c, err := NewDlaaSClient()
		if err != nil {
			cmd.ui.Failed(err.Error())
		}
		params := models.NewGetModelParams().
				WithModelID(modelID).
				WithTimeout(defaultOpTimeout)

		modelInfo, err := c.Models.GetModel(params, BasicAuth())

		if err != nil {
			var s string
			switch err.(type) {
			case *models.GetModelUnauthorized:
				s = badUsernameOrPWD
			case *models.GetModelNotFound:
				s = "Model not found."
			}
			responseError(s, err, cmd.ui)
			return nil
		}

		if isJSON {
			jbytes, err := json.Marshal(modelInfo)
			if err != nil {
				cmd.ui.Failed(err.Error())
			}
			var out bytes.Buffer
			json.Indent(&out, jbytes, "", "\t")
			out.WriteTo(os.Stdout)
		} else {

			m := modelInfo.Payload

			cmd.ui.Say("Id: %s", terminal.EntityNameColor(m.ModelID))
			cmd.ui.Say("Model definition:")
			cmd.ui.Say("  Name: %s", m.Name)
			cmd.ui.Say("  Description: %s", m.Description)
			cmd.ui.Say("  Framework: %s:%s", m.Framework.Name, m.Framework.Version)

			cmd.ui.Say("Training:")
			cmd.ui.Say("  Status: %s", terminal.EntityNameColor(m.Training.TrainingStatus.Status))
			cmd.ui.Say("  Submitted: %s", formatTimestamp(m.Training.TrainingStatus.Submitted))
			cmd.ui.Say("  Completed: %s", formatTimestamp(m.Training.TrainingStatus.Completed))
			learners := m.Training.Learners
			if learners == 0 {
				learners = 1
			}
			cmd.ui.Say("  Resources: %.2f CPUs | %.2f GPUs | %.2f %s Mem | %d node(s)", m.Training.Cpus, m.Training.Gpus,
				m.Training.Memory, *m.Training.MemoryUnit, learners)
			cmd.ui.Say("  Command: %s", strings.TrimSpace(m.Training.Command))
			cmd.ui.Say("  Input data : %s", strings.Join(m.Training.InputData, ","))
			cmd.ui.Say("  Output data: %s", strings.Join(m.Training.OutputData, ","))

			cmd.ui.Say("Data stores:")
			for _, ds := range m.DataStores {
				cmd.ui.Say("  ID: %s", ds.DataStoreID)
				cmd.ui.Say("  Type: %s", ds.Type)
				cmd.ui.Say("  Connection: ")
				for k, v := range ds.Connection {
					cmd.ui.Say("    %s: %s", k, v)
				}
			}
			cmd.ui.Say("Summary metrics:")
			for _, mr := range m.Metrics {
				cmd.ui.Say("  Type: %s {", mr.Type)
				cmd.ui.Say("  	Iteration: %d", mr.Iteration)
				cmd.ui.Say("  	Timestamp: %s", mr.Timestamp)
				for k, v := range mr.Values {
					cmd.ui.Say("  	%s: %v", k, v)
				}
				cmd.ui.Say("  }")
			}

			cmd.ui.Ok()
		}
	}

	return nil
}
