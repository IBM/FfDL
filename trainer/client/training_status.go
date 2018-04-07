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

package client

import (
	"encoding/json"
	"strings"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
)

// TrainingStatusUpdate captures the details for training status update events
type TrainingStatusUpdate struct {
	Status grpc_trainer_v2.Status
	Timestamp float64
	ErrorCode string
	StatusMessage string
}

const (
	// ErrCodeNormal indicates a normal non-error situation
	ErrCodeNormal                 = "000"

	// SERVER ERRORS

	// ErrCodeInsufficientResources indicates a scheduling error due to resource constraints
	ErrCodeInsufficientResources  = "S100"
	// ErrCodeFailedDeploy indicates a general deployment error
	ErrCodeFailedDeploy           = "S101"
	// ErrCodeFailedPS ...
	ErrCodeFailedPS               = "S102"
	// ErrCodeImagePull indicates an image pull error
	ErrCodeImagePull              = "S103"
	// ErrFailedPodReasonUnknown indicates an unknown pod error
	ErrFailedPodReasonUnknown     = "S104"
	// ErrCodeK8SConnection indicates a kubernetes connection error
	ErrCodeK8SConnection          = "S200"
	// ErrCodeEtcdConnection indicates a etcd connection error
	ErrCodeEtcdConnection         = "S201"
	// ErrCodeFailLoadModel indicates an error while loading the model code
	ErrCodeFailLoadModel          = "S301"
	// ErrCodeFailLoadData indicates an error while loading the training data
	ErrCodeFailLoadData           = "S302"
	// ErrCodeFailStoreResults indicates an error while storing the trained model and logs
	ErrCodeFailStoreResults       = "S303"
	// ErrCodeFailStoreResultsOnFail indicates an error while storing the logs on job error
	ErrCodeFailStoreResultsOnFail = "S304"
	// ErrCodeFailStoreResultsOnHalt indicates an error while storing the logs on job halt
	ErrCodeFailStoreResultsOnHalt = "S305"

	// CLIENT ERRORS

	// ErrInvalidManifestFile indicates an invalid manifest file
	ErrInvalidManifestFile    = "C101"
	// ErrInvalidZipFile indicates an invalid ZIP file
	ErrInvalidZipFile         = "C102"
	// ErrInvalidCredentials indicates an invalid set of credentials
	ErrInvalidCredentials     = "C103"
	// ErrInvalidResourceSpecs indicates invalid resouce specifications
	ErrInvalidResourceSpecs   = "C104"
	// ErrLearnerProcessCrash indicates a crash of the process in the learner container
	ErrLearnerProcessCrash    = "C201"
)


// GetStatus converts between a string and proper DLaaS type of job status updates.
// The value parameter is either a status string (e.g., "PROCESSING"), or a JSON string
// with status and error details, e.g., '{"status":"FAILED","exit_code":"51","status_message":"Error opening ZIP file"}'
func GetStatus(value string, logr *logger.LocLoggingEntry) (*TrainingStatusUpdate) {
	status := value
	statusMessage := service.StatusMessages_NORMAL_OPERATION.String()
	errorCode := ""
	timestamp := 0.0
	if strings.HasPrefix(status, "{") {
		var objmap map[string]*json.RawMessage
		err := json.Unmarshal([]byte(status), &objmap)
		if err != nil {
			logr.WithError(err).Errorf("Unable to parse status JSON: %s", status)
		}
		json.Unmarshal(*objmap["status"], &status)
		json.Unmarshal(*objmap["status_message"], &statusMessage)
		json.Unmarshal(*objmap["error_code"], &errorCode)
		json.Unmarshal(*objmap["timestamp"], &timestamp)
	}
	var updStatus grpc_trainer_v2.Status
	switch status {
	case grpc_trainer_v2.Status_PENDING.String():
		updStatus = grpc_trainer_v2.Status_PENDING
	case grpc_trainer_v2.Status_HALTED.String():
		updStatus = grpc_trainer_v2.Status_HALTED
	case grpc_trainer_v2.Status_FAILED.String():
		updStatus = grpc_trainer_v2.Status_FAILED
	case grpc_trainer_v2.Status_DEPLOY.String():
		updStatus = grpc_trainer_v2.Status_DEPLOY
	case grpc_trainer_v2.Status_DOWNLOADING.String():
		updStatus = grpc_trainer_v2.Status_DOWNLOADING
	case grpc_trainer_v2.Status_PROCESSING.String():
		updStatus = grpc_trainer_v2.Status_PROCESSING
	case grpc_trainer_v2.Status_STORING.String():
		updStatus = grpc_trainer_v2.Status_STORING
	case grpc_trainer_v2.Status_COMPLETED.String():
		updStatus = grpc_trainer_v2.Status_COMPLETED
	}
	result := TrainingStatusUpdate{}
	result.ErrorCode = errorCode
	result.Status = updStatus
	result.StatusMessage = statusMessage
	result.Timestamp = timestamp
	return &result
}
