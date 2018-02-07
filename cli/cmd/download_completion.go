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
	"fmt"
)

// DownloadCmdCompletion provide bash auto completion options
func DownloadCmdCompletion(c *cli.Context) {
	args := c.NArg()
	flags := c.NumFlags()

	if args == 1 && flags == 2 {
		return
	}

	if args == 0 {
		ModelIDCompletion(c)
	}

	if args == 1 && flags == 0 {
		fmt.Println("--definition")
		fmt.Println("--trainedmodel")
		fmt.Println("--filename")
	}
	if flags == 1 && (c.Bool("definition") || c.Bool("trainedmodel")) {
		fmt.Println("--filename")
	}
}
