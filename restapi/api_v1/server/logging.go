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

package server

import (
	log "github.com/sirupsen/logrus"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/models"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/events"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/training_data"
)

func logWithPostModelParams(params models.PostModelParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyModelFilename] = params.Manifest.Header.Filename

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithDeleteModelParams(params models.DeleteModelParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithGetModelParams(params models.GetModelParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithDownloadModelDefinitionParams(params models.DownloadModelDefinitionParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithDownloadTrainedModelParams(params models.DownloadTrainedModelParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithEMetricsParams(params training_data.GetEMetricsParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithLoglinesParams(params training_data.GetLoglinesParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)

	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data[logger.LogkeyTrainingID] = params.ModelID

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithGetLogsParams(params models.GetLogsParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithGetListModelsParams(params models.ListModelsParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithGetMetricsParams(params models.GetMetricsParams) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)

	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithUpdateStatusParams(params models.PatchModelParams)  *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data["status"] = params.Payload.Status
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}




////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////////

func logWithGetEventTypeEndpointsParams(params events.GetEventTypeEndpointsParams) *log.Entry {
	data:= logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data["event type"] = params.EventType
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithCreateEventEndpointParams(params events.CreateEventEndpointParams) *log.Entry {
	data:= logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data["event type"] = params.EventType
	data["endpoint ID"] = params.EndpointID
	data["endpoint type"] = params.CallbackURL.Type
	data["endpoint URL"] = params.CallbackURL.URL
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithDeleteEventEndpointParams(params events.DeleteEventEndpointParams) *log.Entry {
	data:= logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data["event type"] = params.EventType
	data["endpoint ID"] = params.EndpointID
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithGetEventEndpointParams(params events.GetEventEndpointParams) *log.Entry {
	data:= logger.NewDlaaSLogData(logger.LogkeyRestAPIService)
	data[logger.LogkeyTrainingID] = params.ModelID
	data[logger.LogkeyUserID] = getUserID(params.HTTPRequest)
	data["event type"] = params.EventType
	data["endpoint ID"] = params.EndpointID
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}
