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
	"time"

	"github.com/spf13/viper"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/metricsmon"
	"github.com/IBM/FfDL/commons/util"
	"github.com/IBM/FfDL/trainer/trainer"
)

func main() {
	config.InitViper()
	logger.Config()

	port := viper.GetInt(config.PortKey)
	if port == 0 {
		port = 30005 // TODO don't hardcode
	}
	service := trainer.NewService()

	var stopSendingMetricsChannel chan struct{}
	if config.CheckPushGatewayEnabled() {
		stopSendingMetricsChannel = metricsmon.StartMetricsPusher("trainer", 30*time.Second, config.GetPushgatewayURL()) //remove this once pull based metrics are implemented
	}

	util.HandleOSSignals(func() {
		service.StopTrainer()
		if config.CheckPushGatewayEnabled() {
			stopSendingMetricsChannel <- struct{}{}
		}
	})
	service.Start(port, false)
}
