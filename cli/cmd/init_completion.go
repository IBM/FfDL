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
)

// InitCmdCompletion provide bash auto completion options
func InitCmdCompletion(c *cli.Context) {
	n := c.NArg()
	switch n {
	case 2, 4, 6:
		options := []string{"--type", "--version", "--job"}
		for _, v := range options {
			if !contains(c.Args(), v) {
				fmt.Println(v)
			}
		}
	case 3, 5, 7:
		options := []string{"caffe", "torch", "tensorflow"}
		if c.Args()[n-1] == "--type" {
			for _, v := range options {
				fmt.Println(v)
			}
		}
	}
	return
}
