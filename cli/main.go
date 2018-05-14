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

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"

	"github.com/IBM/FfDL/cli/cmd"
	"github.com/IBM/FfDL/cli/metadata"

	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/trace"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
)

const (
	pluginName  = "deep-learning"
	dlNamespace = "dl"
)

var buildstamp string
var githash string

// DlaasPlugin is the struct implementing the DLaaS interface.
type DlaasPlugin struct{}

const commandHelp = "NAME:" + `
   {{.Name}} - {{.Description}}{{with .ShortName}}
` + "ALIAS:" + `
   {{.}}{{end}}
` + "USAGE:" + `
   {{.Usage}}
{{with .Flags}}
` + "OPTIONS:" + `
{{range .}}   {{.}}
{{end}}{{end}}
`

// GetMetadata is called by the CF CLI to get the commands provided by this plugin.
func (c *DlaasPlugin) GetMetadata() plugin.PluginMetadata {

	commands := make([]plugin.Command, len(metadata.Commands))

	for index, command := range metadata.Commands {
		commands[index] = plugin.Command{
			Namespace:   command.Namespace,
			Name:        command.Name,
			Description: command.Description,
			Usage:       command.Usage,
			Flags:       command.PluginFlags,
		}
	}

	return plugin.PluginMetadata{
		Name: pluginName,
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 0,
			Minor: 4,
			Build: 0,
		},
		Namespaces: []plugin.Namespace{
			{
				Name:        dlNamespace,
				Description: "Manage deep learning models on Bluemix",
			},
		},
		Commands: commands,
	}
}

// Run is the entry point for every "dl" cli command.
func (c *DlaasPlugin) Run(context plugin.PluginContext, args []string) {
	ui := terminal.NewStdUI()

	defer func() {
		if l, ok := trace.Logger.(trace.Closer); ok {
			l.Close()
		}
	}()

	defer func() {
		if err := recover(); err != nil {
			exitCode, er := strconv.Atoi(fmt.Sprint(err))
			if er == nil {
				os.Exit(exitCode)
			}

			// FIXME QuietPanic does not exist
			//if err != terminal.QuietPanic {
			//	fmt.Printf("%v\n", err)
			//}

			os.Exit(1)
		}
	}()

	actions := map[string]cli.ActionFunc{
		metadata.ProjectInit: func(c *cli.Context) error {
			return cmd.NewInitCmd(ui, context).Run(c)
		},
		metadata.Train: func(c *cli.Context) error {
			return cmd.NewTrainCmd(ui, context).Run(c)
		},
		metadata.Show: func(c *cli.Context) error {
			return cmd.NewShowCmd(ui, context).Run(c)
		},
		metadata.Delete: func(c *cli.Context) error {
			return cmd.NewDeleteCmd(ui, context).Run(c)
		},
		metadata.List: func(c *cli.Context) error {
			return cmd.NewListCmd(ui, context).Run(c)
		},
		metadata.Download: func(c *cli.Context) error {
			return cmd.NewDownloadCmd(ui, context).Run(c)
		},
		metadata.Logs: func(c *cli.Context) error {
			return cmd.NewLogsCmd(ui, context).Run(c)
		},
		metadata.Loglines: func(c *cli.Context) error {
			return cmd.NewLoglinesCmd(ui, context).Run(c)
		},
		metadata.Emetrics: func(c *cli.Context) error {
			return cmd.NewEmetricsCmd(ui, context).Run(c)
		},
		metadata.Halt: func(c *cli.Context) error {
			return cmd.NewHaltCmd(ui, context).Run(c)
		},
		metadata.Version: func(c *cli.Context) error {
			return cmd.NewVersion(ui, context).Run(c)
		},
	}

	bashCompletes := map[string]cli.BashCompleteFunc{
		metadata.Delete:    	cmd.ModelIDCompletion,
		metadata.Train:     	cmd.TrainCmdCompletion,
		metadata.Show:       	cmd.ModelIDCompletion,
		metadata.ProjectInit: cmd.InitCmdCompletion,
		metadata.Download: 		cmd.DownloadCmdCompletion,
		metadata.Logs:    		cmd.TrainingLogsCompletion,
		metadata.Loglines:    	cmd.LoglinesCompletion,
		metadata.Emetrics:    	cmd.EMetricsCompletion,
		metadata.Halt:    		cmd.ModelIDCompletion,
	}

  cli.CommandHelpTemplate = commandHelp
	cli.BashCompletionFlag = cli.BoolFlag{
		Name:   "generate-dl-completion",
		Hidden: true,
	}

	app := cli.NewApp()
	app.Name = "bluemix dl"
	app.EnableBashCompletion = true
	app.Commands = make([]cli.Command, len(metadata.Commands))
	for index, command := range metadata.Commands {
		app.Commands[index] = cli.Command{
			Name:         command.Name,
			Description:  command.Description,
			Usage:        command.Usage,
			Flags:        command.CliFlags,
			Action:       actions[command.Name],
			BashComplete: bashCompletes[command.Name],
		}
	}

	app.Run(os.Args)
}

func main() {
	cmd.SetBuildstamp(buildstamp)
	cmd.SetGithash(githash)
	plugin.Start(new(DlaasPlugin))
}
