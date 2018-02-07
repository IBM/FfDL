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
	"github.ibm.com/ffdl/ffdl-core/restapi/api_v1/client/models"
)

// ModelIDCompletion provide bash auto completion options for a model ID.
func ModelIDCompletion(c *cli.Context) {
	n := c.NArg()
	if n > 1 {
		return
	}

	client, err := NewDlaaSClient()
	if err != nil {
		return
	}

	params := models.NewListModelsParams().
		WithTimeout(defaultOpTimeout)

	modelz, err := client.Models.ListModels(params, BasicAuth())
	if err != nil {
		return
	}

	for _, v := range modelz.Payload.Models {
		fmt.Println(v.ModelID)
	}
	return
}
