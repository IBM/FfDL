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

package lcm

import (
	log "github.com/sirupsen/logrus"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
)

func (s *lcmService) logWithTrainerContainerInfosRequest(req *service.TrainerContainerInfosRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId
	data[logger.LogkeyIsFollow] = req.Follow
	data[logger.LogkeyIsMetrics] = req.Metrics
	data[logger.LogkeyIsSummary] = req.Summary

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithJobDeploymentRequest(req *service.JobDeploymentRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithJobKillRequest(req *service.JobKillRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithJobHaltRequest(req *service.JobHaltRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithFields(fields log.Fields) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)

	for k, v := range fields {
		data[k] = v
	}

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithCreateEventEndpointRequest(req *service.CreateEventEndpointRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId
	data["event type"] = req.EventType
	data["endpoint url"] = req.EndpointUrl
	data["endpoint type"] = req.EndpointType
	data["endpoint id"] = req.EndpointId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithGetEventTypeEndpointsRequest(req *service.GetEventTypeEndpointsRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId
	data["event type"] = req.EventType

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func (s *lcmService) logWithDeleteEventEndpointRequest(req *service.DeleteEventEndpointRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId
	data["event type"] = req.EventType
	data["endpoint id"] = req.EndpointId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}
func (s *lcmService) logWithGetEventEndpointRequest(req *service.GetEventEndpointRequest) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyLcmService)
	data[logger.LogkeyTrainingID] = req.TrainingId
	data[logger.LogkeyUserID] = req.UserId
	data["event type"] = req.EventType
	data["endpoint id"] = req.EndpointId

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}
