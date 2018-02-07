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
	"time"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client/training_data"
	dlaasClient "github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client"
	"encoding/json"
)

// EmetricsCmd represents the instance of this command
type EmetricsCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewEmetricsCmd creates a new instance of this command
func NewEmetricsCmd(ui terminal.UI, context plugin.PluginContext) *EmetricsCmd {
	return &EmetricsCmd{
		ui:      ui,
		context: context,
	}
}

func printEMetrics(cmd *EmetricsCmd, tdc *dlaasClient.Dlaas, params *training_data.GetEMetricsParams, isJSON bool) (int64, int, error) {
	emetrics, err := tdc.TrainingData.GetEMetrics(params, BasicAuth())
	if err != nil {
		cmd.ui.Failed("Could not read emetrics: %s", err.Error())
		return 0, 0, err
	}

	var lastTimestamp int64
	for _, metrics := range emetrics.Payload.Models {
		lastTimestamp = metrics.Meta.Time
		if isJSON {
			jsonBytes, err := json.Marshal(metrics)
			if err != nil {
				cmd.ui.Failed("Could not marshal record to json: %s", err.Error())
				return 0, 0, err
			}
			fmt.Printf("%s\n", string(jsonBytes))
		} else {
			fmt.Printf("time: %d, group-label: %s, training-id: %s\n",
				metrics.Meta.Time, metrics.Grouplabel, metrics.Meta.TrainingID)

			//var etimes map[string]*trainingDataClient.Any
			etimes := metrics.Etimes
			fmt.Printf("    etimes: ")
			for k, v := range etimes {
				fmt.Printf("%s: %s, ", k, v)
			}
			fmt.Printf("\n")

			fmt.Printf("    values: ")
			//var values map[string]*trainingDataClient.Any
			values := metrics.Values

			for k, v := range values {
				fmt.Printf("%s: %s, ", k, v)
			}
			fmt.Printf("\n")
		}
	}
	return lastTimestamp, len(emetrics.Payload.Models), nil
}

// Run is the handler for the emetrics CLI command.
func (cmd *EmetricsCmd) Run(cliContext *cli.Context) error {
	log.SetLevel(log.WarnLevel)
	cmd.config = cmd.context.PluginConfig()

	args := cliContext.Args()

	if len(args) == 0 {
		cmd.ui.Failed("Incorrect arguments")
		return nil
	}
	trainingID := args[0]

	isFollow := cliContext.IsSet("follow")
	isJSON := cliContext.IsSet("json")

	pagesize := int32(cliContext.Int("pagesize"))
	if pagesize == 0 {
		pagesize = defaultLogsPageSize
	}

	pos := int64(cliContext.Int("pos"))

	since := cliContext.String("since")
	log.Debugf("since: %s", since)

	tdc, err := NewDlaaSClient()
	if err != nil {
		cmd.ui.Failed(err.Error())
		return nil
	}

	params := training_data.NewGetEMetricsParamsWithTimeout(defaultOpTimeout)

	params.ModelID = trainingID

	params.Pagesize = &pagesize
	params.Pos = &pos
	params.SinceTime = &since
	searchType := "TERM"
	params.SearchType = &searchType

	lastTimestamp, nPrinted, err := printEMetrics(cmd, tdc, params, isJSON)
	if err != nil {
		cmd.ui.Failed("Could not read emetrics: %s", err.Error())
		return nil
	}
	totalPrinted := int32(nPrinted)
	for ; totalPrinted < pagesize; {
		sinceTimeQuery := fmt.Sprintf("%v", lastTimestamp+1)
		params.SinceTime = &sinceTimeQuery
		log.Debugf("requesting emetrics past %s", sinceTimeQuery)

		lastTimestamp, nPrinted, err = printEMetrics(cmd, tdc, params, isJSON)

		if err != nil {
			cmd.ui.Failed("Could not read emetrics: %s", err.Error())
			return nil
		}
		if nPrinted == 0 {
			break
		}
		totalPrinted += int32(nPrinted)
	}

	if isFollow {
		time.Sleep(defaultLogsFollowSleep)

		for {
			sinceTimeQuery := fmt.Sprintf("%v", lastTimestamp+1)
			params.SinceTime = &sinceTimeQuery
			log.Debugf("requesting emetrics past %s (follow)", sinceTimeQuery)

			lastTimestamp, _, err = printEMetrics(cmd, tdc, params, isJSON)

			if err != nil {
				cmd.ui.Failed("Could not read emetrics: %s", err.Error())
				return nil
			}
		}
	}

	return nil
}
