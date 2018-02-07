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
	"github.com/urfave/cli"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/bluemix/terminal"
	"github.com/IBM-Bluemix/bluemix-cli-sdk/plugin"
)

var buildstamp string
var githash string

// Version is the struct to show model git hash and build time
type Version struct {
	ui      terminal.UI
	config  plugin.PluginConfig
	context plugin.PluginContext
}

// NewVersion creates a new instance of this command
func NewVersion(ui terminal.UI, context plugin.PluginContext) *Version {
	return &Version{
		ui:      ui,
		context: context,
	}
}

// Run is the handler for the version CLI command.
func (cmd *Version) Run(cliContext *cli.Context) error {
	cmd.config = cmd.context.PluginConfig()
	s := fmt.Sprintf("Git Commit Hash: %s", githash)
	cmd.ui.Say(s)
	s1 := fmt.Sprintf("UTC Build Time : %s", buildstamp)
	cmd.ui.Say(s1)
	return nil
}

// SetBuildstamp sets the build stamp on this component
func SetBuildstamp(bstamp string) {
	buildstamp = bstamp
}

// SetGithash set the git hash on this component
func SetGithash(ghash string) {
	githash = ghash
}
