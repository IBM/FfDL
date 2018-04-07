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

package jobmonitor

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	lcmClient "github.com/IBM/FfDL/commons/service/client"
	trainerClient "github.com/IBM/FfDL/trainer/client"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
	"golang.org/x/net/context"
)


func prepareStatusMessage(errorCode string, statusMessage string) (string) {
	if errorCode == trainerClient.ErrCodeFailLoadModel {
		return fmt.Sprintf("Unable to load model (exit code: '%s')", statusMessage)
	} else if errorCode == trainerClient.ErrCodeFailLoadData {
		return fmt.Sprintf("Unable to load training data (exit code: '%s')", statusMessage)
	} else if errorCode == trainerClient.ErrCodeFailStoreResults {
		return fmt.Sprintf("Unable to store results (exit code: '%s')", statusMessage)
	} else if errorCode == trainerClient.ErrCodeFailStoreResultsOnFail {
		return fmt.Sprintf("Unable to store results on failure (exit code: '%s')", statusMessage)
	} else if errorCode == trainerClient.ErrCodeFailStoreResultsOnHalt {
		return fmt.Sprintf("Unable to store results on halt (exit code: '%s')", statusMessage)
	} else if errorCode == trainerClient.ErrLearnerProcessCrash {
		return fmt.Sprintf("Learner process terminated with an error (exit code: '%s')", statusMessage)
	}
	return errorCode
}

func sendStatusUpdate(trainingID string, status string, logr *logger.LocLoggingEntry) error {
	message := `{"trainingId":"` + trainingID + `","status":"` + status + `"}`
	return sendToEndpoints(message, trainingID, "status", logr)
}

// guaranteed to be sent new metrics
func metricsUpdate(trainingID string, metrics *grpc_trainer_v2.Metrics, logr *logger.LocLoggingEntry) error {
	values := `{`
	first := true
	for k, v := range metrics.Values {
		if first {
			values += `"` + k + `":"` + v + `"`
			first = false
			continue
		}
		values += `,"` + k + `":"` + v + `"`
	}

	values = values + `}`

	message := `{"timestamp": "` + metrics.Timestamp + `", "type": "` + metrics.Type + `", "iteration": "` + fmt.Sprint(metrics.Iteration) + `", "values": ` + values + `}`

	message = `{"trainingId":"` + trainingID + `", "metrics":` + message + `}`

	return sendToEndpoints(message, trainingID, "metrics", logr)
}

func sendToEndpoints(message string, trainingID string, eventType string, logr *logger.LocLoggingEntry) error {
	logr.Debugf("(sendToEndpoints) Sending " + eventType + " notifications")

	lcm, err := lcmClient.NewLcm(nil)
	if err != nil {
		return err
	}

	getEndpointsReq := &service.GetEventTypeEndpointsRequest{TrainingId: trainingID, UserId: "", EventType: eventType}
	res, err := lcm.Client().GetEventTypeEndpoints(context.Background(), getEndpointsReq)

	getEndpointsAllReq := &service.GetEventTypeEndpointsRequest{TrainingId: trainingID, UserId: "", EventType: "all"}
	resAll, errAll := lcm.Client().GetEventTypeEndpoints(context.Background(), getEndpointsAllReq)

	if err != nil && errAll != nil {
		return err
	}
	if res == nil && resAll == nil {
		return errors.New("No endpoints of type '" + eventType + "' for training id '" + trainingID + "'")
	}

	endpoints := []*service.Endpoint{}
	if res != nil {
		endpoints = append(endpoints, res.Endpoints...)
	}
	if resAll != nil {
		endpoints = append(endpoints, resAll.Endpoints...)
	}

	sent := []string{}

	for _, e := range endpoints {
		url := e.Url
		if contains(sent, url) {
			continue
		}
		sent = append(sent, url)

		var jsonStr = []byte(message)

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			logr.WithError(err).Debugf("(sendToEndpoints) Failed to send status update to: %s", url)
			continue
		}
		defer resp.Body.Close()

		logr.Debugf("(sendToEndpoints) Response Status '%s' from url : %s", resp.Status, url)

		continue
	}
	return nil
}

func contains(array []string, element string) bool {
	for _, s := range array {
		if s == element {
			return true
		}
	}
	return false
}
