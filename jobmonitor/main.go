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
	"strconv"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/metricsmon"
	"github.com/IBM/FfDL/commons/util"
	jobM "github.com/IBM/FfDL/jobmonitor/jobmonitor"

	"os"
	"time"
)

func main() {
	config.InitViper()
	logger.Config()

	statsdClient := metricsmon.NewStatsdClient("jobmonitor")
	if config.CheckPushGatewayEnabled() {
		metricsmon.StartStatsdMetricsPusher(statsdClient, 10*time.Second)
	}
	useNativeDistribution, _ := strconv.ParseBool(os.Getenv("USE_NATIVE_DISTRIBUTION"))
	numLearners, _ := strconv.Atoi(os.Getenv("NUM_LEARNERS"))
	trainingID := os.Getenv("TRAINING_ID")
	userID := os.Getenv("USER_ID")
	jobName := os.Getenv("JOB_NAME")

	logr := logger.LocLogger(jobM.InitLogger(trainingID, userID))
	jm, err := jobM.NewJobMonitor(trainingID, userID, numLearners, jobName, useNativeDistribution, statsdClient, logr)

	if err != nil {
		logr.WithError(err).Errorf("failed to bring up job monitor for training %s, already must have signaled to kill the jm", trainingID)
	} else {
		logr.Infof("Job Monitor instantiated and ready to go. Starting to manage %s", jm.TrainingID)

		go jm.ManageDistributedJob(logr)

		util.HandleOSSignals(func() {
			logr.Warningln(" ###### shutting down job monitor ###### ")
			jm.EtcdClient.Close(logr)

		})

		//This seems to be the only way to prevent the container from exiting.
		//JobMonitor is not a service. In the LCM we can use service.Start() to keep the container from exiting.
		for true {
			time.Sleep(600 * time.Second)
		}
	}

}
