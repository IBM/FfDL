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
	"github.com/spf13/viper"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/util"
	"github.com/IBM/FfDL/metrics/service"
)

func main() {
	config.InitViper()
	service.InitViper()
	logger.Config()

	logr := logger.LocLogger(logger.LogServiceBasic(service.LogkeyTrainingDataService))
	logr.Debugf("function entry")

	port := viper.GetInt(config.PortKey)
	if port == 0 {
		port = 30015 // TODO don't hardcode
	}
	logr.Debugf("Port is: %d", port)

	logr.Debugf("Creating dlaas-training-metrics-service")

	service := service.NewService()
	util.HandleOSSignals(func() {
		service.Stop()
	})
	logr.Debugf("Calling  service.Start on dlaas-training-metrics-service")
	service.Start(port, false)

	logr.Debugf("function exit")
}
