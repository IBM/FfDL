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

const (
	defaultLogsPageSize = 20
	defaultLogsFollowSleep = time.Second*2
)

// LoglinesCmd represents the instance of this command
type LoglinesCmd struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewLoglinesCmd creates a new instance of this command
func NewLoglinesCmd(ui terminal.UI, context plugin.PluginContext) *LoglinesCmd {
	return &LoglinesCmd{
		ui:      ui,
		context: context,
	}
}

func printLoglines(cmd *LoglinesCmd, tdc *dlaasClient.Dlaas, params *training_data.GetLoglinesParams, isJSON bool) (int64, int, error) {
	loglines, err := tdc.TrainingData.GetLoglines(params, BasicAuth())
	if err != nil {
		cmd.ui.Failed("Could not read loglines: %s", err.Error())
		return 0, 0, err
	}

	var lastTimestamp int64
	for _, logRecord := range loglines.Payload.Models {
		lastTimestamp = logRecord.Meta.Time
		if isJSON {
			jsonBytes, err := json.Marshal(logRecord)
			if err != nil {
				cmd.ui.Failed("Could not marshal record to json: %s", err.Error())
				return 0, 0, err
			}
			fmt.Printf("%s\n", string(jsonBytes))
		} else {
			fmt.Printf("%v %v %s\n", logRecord.Meta.Rindex, logRecord.Meta.Time, logRecord.Line)
		}
	}
	return lastTimestamp, len(loglines.Payload.Models), nil
}


// Run is the handler for the loglines CLI command.
func (cmd *LoglinesCmd) Run(cliContext *cli.Context) error {
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
		return nil
	}

	pos := int64(cliContext.Int("pos"))

	since := cliContext.String("since")
	log.Debugf("since: %s", since)

	tdc, err := NewDlaaSClient()
	if err != nil {
		cmd.ui.Failed(err.Error())
		return nil
	}

	params := training_data.NewGetLoglinesParamsWithTimeout(defaultOpTimeout)

	params.ModelID = trainingID

	params.Pagesize = &pagesize
	params.Pos = &pos
	params.SinceTime = &since
	searchType := "TERM"
	params.SearchType = &searchType

	lastTimestamp, nPrinted, err := printLoglines(cmd, tdc, params, isJSON)

	if err != nil {
		cmd.ui.Failed("Could not read log lines: %s", err.Error())
		return nil
	}
	totalPrinted := int32(nPrinted)
	for ; totalPrinted < pagesize; {
		sinceTimeQuery := fmt.Sprintf("%v", lastTimestamp+1)
		params.SinceTime = &sinceTimeQuery
		log.Debugf("requesting logs past %s", sinceTimeQuery)

		lastTimestamp, nPrinted, err = printLoglines(cmd, tdc, params, isJSON)

		if err != nil {
			cmd.ui.Failed("Could not read log lines: %s", err.Error())
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
			log.Debugf("requesting logs past (follow) %s", sinceTimeQuery)

			lastTimestamp, _, err = printLoglines(cmd, tdc, params, isJSON)

			if err != nil {
				cmd.ui.Failed("Could not read emetrics: %s", err.Error())
				return nil
			}
		}
	}

	return nil
}
