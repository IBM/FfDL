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
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
)

// TrainCmdCompletion provide bash auto completion options
func TrainCmdCompletion(c *cli.Context) {
	n := c.NArg()
	if n >= 2 {
		return
	}
	switch n {
	case 0:
		var manifestFiles []string
		err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			if err != nil || f.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
				manifestFiles = append(manifestFiles, path)
			}
			return nil
		})
		if err != nil {
			return
		}
		for _, v := range manifestFiles {
			fmt.Println(v)
		}
	case 1:
		var modelDefinitionFiles []string
		err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return nil
			} else if f.IsDir() {
				modelDefinitionFiles = append(modelDefinitionFiles, path)
			} else if strings.HasSuffix(path, "zip") {
				modelDefinitionFiles = append(modelDefinitionFiles, path)
			}
			return nil
		})
		if err != nil {
			return
		}
		for _, v := range modelDefinitionFiles {
			fmt.Println(v)
		}
	}

	return
}
