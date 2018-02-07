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

package trainer

import (
	"fmt"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
	"strings"
	"github.ibm.com/ffdl/ffdl-core/commons/config"
)

func validateFrameworks(fw *grpc_trainer_v2.Framework) (bool, string) {
	fwName := normalizedFrameworkName(fw)
	fwVersion := fw.Version

	if fwName == "" {
		return false, "framework name is required"
	}

	if fwVersion == "" {
		return false, "framework version is required"
	}

	loc := config.GetCurrentLearnerConfigLocation(fwName, fwVersion)
	if loc == "" {
		return false, fmt.Sprintf("%s version %s not supported", fwName, fwVersion)
	}
	return true, ""
}

func normalizedFrameworkName(fw *grpc_trainer_v2.Framework) string {
	return strings.ToLower(fw.Name)
}
