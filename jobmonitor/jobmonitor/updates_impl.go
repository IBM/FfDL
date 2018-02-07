package jobmonitor

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	lcmClient "github.ibm.com/ffdl/ffdl-core/commons/service/client"
	trainerClient "github.ibm.com/ffdl/ffdl-core/trainer/client"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
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
