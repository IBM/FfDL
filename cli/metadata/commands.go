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

package metadata

import (
	"github.com/urfave/cli"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
)

const (
	deepLearningNS = "dl"

	// ProjectInit is the name of CLI command to create a project template.
	ProjectInit = "init"

	// Train is the name of the CLI command to deploy a model.
	Train = "train"

	// Show is the name of the CLI command to get a model's info.
	Show = "show"

	// Delete is the name of the CLI command to delete a model.
	Delete = "delete"

	// List is the name of the CLI command to list all models.
	List = "list"

	// Download is the name of the CLI command to get the original model definition as ZIP.
	Download = "download"

	// Halt is the name of the CLI command to get a training job's info.
	Halt = "halt"

	// Logs is the name of the CLI command to get the training logs. (deprecated)
	Logs = "logs"

	// Loglines is the name of the CLI command to get the training logs.
	Loglines = "loglines"

	// Emetrics is the name of the CLI command to get the evaluation metrics.
	Emetrics = "emetrics"

	// Version is the version CLI command.
	Version = "version"
)

type commandMetadata struct {
	Namespace   string
	Name        string
	Description string
	Usage       string
	PluginFlags []plugin.Flag // Used by Bluemix CLI
	CliFlags    []cli.Flag    // Used by codegangsta/CLI
	Action      func(c *cli.Context)
}

var (
	// Commands are a description of the DLaaS CLI commands.
	Commands = []*commandMetadata{
		{
			Namespace:   deepLearningNS,
			Name:        ProjectInit,
			Description: "Creates a new deep learning project with a manifest.",
			Usage:       "bx dl init NAME",
			PluginFlags: []plugin.Flag{
			},
			CliFlags: []cli.Flag{
				cli.StringFlag{
					Name:  "type",
					Usage: "Deep learning framework (e.g. Caffe, Torch).",
				},
				cli.StringFlag{
					Name:  "version",
					Usage: "Deep learning framework version.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Train,
			Description: "Trains a model",
			Usage:       "bx dl train MANIFEST_FILE (MODEL_DEFINITION_ZIP|MODEL_DEFINITION_DIR)",
			PluginFlags: []plugin.Flag{},
			CliFlags:    []cli.Flag{},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Show,
			Description: "Get detailed information about a model and training status",
			Usage:       "bx dl show MODEL_ID",
			PluginFlags: []plugin.Flag{
				{
					Name:        "json",
					HasValue:    false,
					Description: "If specified, output as json",
				},
			},
			CliFlags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "json",
					Usage: "If specified, output as json.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Delete,
			Description: "Delete a model",
			Usage:       "bx dl delete MODEL_ID",
			PluginFlags: []plugin.Flag{},
			CliFlags:    []cli.Flag{},
		},
		{
			Namespace:   deepLearningNS,
			Name:        List,
			Description: "List all models",
			Usage:       "bx dl list",
			PluginFlags: []plugin.Flag{},
			CliFlags:    []cli.Flag{},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Download,
			Description: "Download the model definition as ZIP file",
			Usage:       "bx dl download MODEL_ID (--definition|--trainedmodel) [--filename FILENAME]",
			PluginFlags: []plugin.Flag{
				{
					Name:        "definition",
					Description: "Download the model definition.",
				},
				{
					Name:        "trainedmodel",
					Description: "Download the trained model.",
				},
				{
					Name:        "filename",
					Description: "Filename of the downloaded ZIP file.",
				},
			},
			CliFlags: []cli.Flag{
				cli.BoolFlag{
					Name: "definition",
					Usage: "Download the model definition.",
				},
				cli.BoolFlag{
					Name: "trainedmodel",
					Usage: "Download the trained model.",
				},
				cli.StringFlag{
					Name:  "filename",
					Usage: "Filename of the downloaded ZIP file.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Logs,
			Description: "View stream of logs (deprecated, consider using loglines or emetrics instead)",
			Usage:       "bx dl logs MODEL_ID [--follow] [--metrics]",
			PluginFlags: []plugin.Flag{
				{
					Name:        "follow",
					HasValue:    false,
					Description: "If specified, follow the log",
				},
				{
					Name:        "metrics",
					HasValue:    false,
					Description: "If specified, deliver parsed evaluation metrics",
				},
				{
					Name:        "json",
					HasValue:    false,
					Description: "If specified, output metrics as json",
				},
			},
			CliFlags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "follow",
					Usage: "If specified, follow the log.",
				},
				cli.BoolTFlag{
					Name:  "metrics",
					Usage: "If specified, deliver parsed evaluation metrics.",
				},
				cli.BoolTFlag{
					Name:  "json",
					Usage: "If specified, output metrics as json.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Loglines,
			Description: "View log lines",
			Usage:       "bx dl loglines MODEL_ID [--follow] [--metrics]",
			PluginFlags: []plugin.Flag{
				{
					Name:        "follow",
					HasValue:    false,
					Description: "If specified, follow the log",
				},
				{
					Name:        "json",
					HasValue:    false,
					Description: "If specified, output metrics as json",
				},
				{
					Name:        "pagesize",
					HasValue:    true,
					Description: "Number of lines to deliver",
				},
				{
					Name:        "pos",
					HasValue:    true,
					Description: "If positive, line number from start, if negative, line position from end",
				},
				{
					Name:        "since",
					HasValue:    true,
					Description: "Only logs after the time (Unix timestamp)",
				},
			},
			CliFlags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "follow",
					Usage: "If specified, follow the log.",
				},
				cli.BoolTFlag{
					Name:  "json",
					Usage: "If specified, output metrics as json.",
				},
				cli.IntFlag{
					Name:  "pagesize",
					Usage: "Number of lines to deliver.",
				},
				cli.IntFlag {
					Name:        "pos",
					Usage: "If positive, line number from start, if negative, line position from end",
				},
				cli.StringFlag{
					Name:  "since",
					Usage: "Only logs after the time.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Emetrics,
			Description: "View evaluation metrics",
			Usage:       "bx dl emetrics MODEL_ID [--follow] [--metrics]",
			PluginFlags: []plugin.Flag{
				{
					Name:        "follow",
					HasValue:    false,
					Description: "If specified, follow the log",
				},
				{
					Name:        "json",
					HasValue:    false,
					Description: "If specified, output metrics as json",
				},
				{
					Name:        "pagesize",
					HasValue:    true,
					Description: "Number of lines to deliver",
				},
				{
					Name:        "pos",
					HasValue:    true,
					Description: "If positive, line number from start, if negative, line position from end",
				},
				{
					Name:        "since",
					HasValue:    true,
					Description: "Only logs after the time (Unix timestamp)",
				},
			},
			CliFlags: []cli.Flag{
				cli.BoolTFlag{
					Name:  "follow",
					Usage: "If specified, follow the log.",
				},
				cli.BoolTFlag{
					Name:  "json",
					Usage: "If specified, output metrics as json.",
				},
				cli.IntFlag{
					Name:  "pagesize",
					Usage: "Number of lines to deliver.",
				},
				cli.IntFlag {
					Name:        "pos",
					Usage: "If positive, line number from start, if negative, line position from end",
				},
				cli.StringFlag{
					Name:  "since",
					Usage: "Only logs after the time.",
				},
			},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Halt,
			Description: "Halt a training job",
			Usage:       "bx dl halt MODEL_ID",
			PluginFlags: []plugin.Flag{},
			CliFlags:    []cli.Flag{},
		},
		{
			Namespace:   deepLearningNS,
			Name:        Version,
			Description: "show git hash and build time of cli",
			Usage:       "bx dl version",
			PluginFlags: []plugin.Flag{},
			CliFlags:    []cli.Flag{},
		},
	}
)
