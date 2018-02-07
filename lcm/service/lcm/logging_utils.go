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
	"encoding/json"
	"fmt"
	"time"

	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	"github.ibm.com/ffdl/ffdl-core/trainer/client"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"

	v1core "k8s.io/api/core/v1"
)

func sendErrorMsg(logr *logger.LocLoggingEntry, outStream service.LifecycleManager_GetTrainingLogStreamServer, sendIt bool, err error) error {
	if sendIt {
		errSend := outStream.Send(&service.TrainerLogStreamResponse{
			Data: []byte(err.Error()),
		})
		if errSend != nil {
			logr.WithError(errSend).Error("Can't send error message to stream.")
		}
	}
	return err
}

// MetricData contains logging metrics, encoded in a generic key/values map.
//
// WARNING: Copied from dlaas-rest-client/go/restmodels/metric_data.go.  Copying it here because this is,
// apparently, the only backwards explicit dependency.  Though,I suspect there are other, implicit dependencies.
// TODO:
// This should really be fixed by specifying via gRPC what the contract is!!! But table this until
// 	issue: As a DLaaS user, I want generic key/value pairs for metric values, so I can output what I like #1371
// 	(https://github.ibm.com/deep-learning-platform/dlaas/issues/1371)
// is resolved.
type MetricData struct {

	// map of key/values, that describe evaluation metrics
	Values map[string]interface{} `json:"Values,omitempty"`

	// Current iteration number be processed.
	Iteration int32 `json:"iteration,omitempty"`

	// Timestamp of the metric. Format: yyyy-MM-dd'T'HH:mm:ss.SSS'Z'
	//
	Timestamp string `json:"timestamp,omitempty"`

	// The type of metrics data
	Type string `json:"type,omitempty"`
}

func sendStatusMessage(logr *logger.LocLoggingEntry, outStream service.LifecycleManager_GetTrainingLogStreamServer,
	bytebuf string, isMetrics bool) error {

	if isMetrics {
		t := time.Now().UTC()
		timeBytes, err := t.MarshalText()
		if err != nil {
			logr.WithError(err).Debug("Time construction error")
			return err
		}
		timeStr := string(timeBytes)

		valuesMap := make(map[string]interface{})
		valuesMap["Message"] = string(bytebuf)
		metricsEntry := MetricData{
			Type:      "Status",
			Values:    valuesMap,
			Timestamp: timeStr,
		}
		jsonBytes, err := json.Marshal(metricsEntry)
		if err != nil {
			logr.WithError(err).Error("Can't construct message")
			return err
		}
		// Would be nice if there was a cheaper way to append a newline
		jsonBytesWithNewLine := []byte(fmt.Sprintf("%s\n",
			string(jsonBytes)))

		errSend := outStream.Send(&service.TrainerLogStreamResponse{
			Data: jsonBytesWithNewLine,
		})
		if errSend != nil {
			logr.WithError(errSend).Error("Can't send error message to stream.")
		}
	} else {
		errSend := outStream.Send(&service.TrainerLogStreamResponse{
			Data: []byte(fmt.Sprintf("Status: %s\n", bytebuf)),
		})
		if errSend != nil {
			logr.WithError(errSend).Error("Can't send error message to stream.")
		}

	}
	return nil
}

// Tell if the log-collector is ready to be used.
func isLogCollectorReady(pods *v1core.PodList, logr *logger.LocLoggingEntry, logContainerName string) bool {
	logr.Debugf("len(pods.Items) is %d", len(pods.Items))
	isReady := false
	nPods := len(pods.Items)
	if nPods > 0 {
		for i := nPods - 1; i >= 0; i-- {
			containerStatuses := pods.Items[i].Status.ContainerStatuses
			for _, containerStatus := range containerStatuses {
				logr.Debugf("Found pod container: %s", containerStatus.Name)
				if containerStatus.Name == logContainerName {
					isReady = containerStatus.Ready
					logr.Debugf("Found the logging container: %s, %t, %s",
						logContainerName, isReady, containerStatus.State.Running)
					// containerStatus.Ready doesn't seem reliable, so just return that it's ready, for
					// lack of better idea.
					isReady = true
					break
				}
			}
		}
	}
	return isReady
}

func getLearnerPodName(pods *v1core.PodList, logr *logger.LocLoggingEntry, learnerName string) string {
	logr.Debugf("len(pods.Items) is %d", len(pods.Items))
	var learnerPodName string
	nPods := len(pods.Items)
	if nPods > 0 {
		for i := nPods - 1; i >= 0; i-- {
			containerStatuses := pods.Items[i].Status.ContainerStatuses
			for _, containerStatus := range containerStatuses {
				logr.Debugf("Found pod container: %s", containerStatus.Name)
				if containerStatus.Name == learnerName {
					learnerPodName = pods.Items[i].Name
					logr.Debugf("Found the learner container: %s, %v",
						learnerPodName, containerStatus.State.Running)
					break
				}
			}
		}
	}
	return learnerPodName
}

func isJobCompleted(logr *logger.LocLoggingEntry, lcm *lcmService, req *service.TrainerContainerInfosRequest,
	shouldConsiderStoringDone bool) (grpc_trainer_v2.Status, bool) {

	path := req.TrainingId + "/" + zkStatus
	response, error := lcm.etcdClient.Get(path, logr)
	if error != nil || response == nil || len(response) == 0 {
		// I don't think it is worth it to look in the persistent store,
		// but I leave open the possibility to think about.
		logr.WithError(error).Errorf("etcd does not have the status, giving up")

		// this things are this messed up, I prefer to give up trying to return results
		return grpc_trainer_v2.Status_COMPLETED, false
	}
	jobStatus := response[0].Value
	logr.Debugf("job status from etcd.getStatus: %s", jobStatus)
	status := client.GetStatus(jobStatus, logr).Status
	if status == grpc_trainer_v2.Status_COMPLETED || status == grpc_trainer_v2.Status_FAILED || status == grpc_trainer_v2.Status_HALTED {
		return status, false
	}
	// This case is a little iffy here, whether the STORING state should be considered in restart, so, use flag.
	if shouldConsiderStoringDone && status == grpc_trainer_v2.Status_STORING {
		return status, false
	}
	return status, true
}
