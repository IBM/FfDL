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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
)

var learnerConfigPath = "/etc/learner-config-json/learner-config.json"

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

func getExternalVersions() (grpc_trainer_v2.Frameworks, error) {
	frameworks, err := getFrameworks()
	if err != nil {
		return grpc_trainer_v2.Frameworks{}, err
	}
	return frameworks, nil
}

func getFrameworks() (grpc_trainer_v2.Frameworks, error) {
	var frameworks grpc_trainer_v2.Frameworks
	fileData, err := readFile(learnerConfigPath)
	if err != nil {
		return grpc_trainer_v2.Frameworks{}, err
	}
	err = json.Unmarshal(fileData, &frameworks)
	if err != nil {
		return grpc_trainer_v2.Frameworks{}, err
	}

	removeInternalFrameworks(frameworks)

	return frameworks, nil
}

func readFile(location string) ([]byte, error) {
	fileData, err := ioutil.ReadFile(location)
	if err != nil {
		return []byte(""), err
	}
	return fileData, nil
}

func removeInternalFrameworks(frameworks grpc_trainer_v2.Frameworks) {
	for frameworkName, frameworkVersion := range frameworks.Frameworks {
		var externalFrameworks []*grpc_trainer_v2.FrameworkDetails
		for _, framework := range frameworkVersion.Versions {
			if framework.External {
				externalFrameworks = append(externalFrameworks, framework)
			}
		}
		if len(externalFrameworks) <= 0 {
			delete(frameworks.Frameworks, frameworkName)
		} else {
			frameworkVersion.Versions = externalFrameworks
		}
	}
}
