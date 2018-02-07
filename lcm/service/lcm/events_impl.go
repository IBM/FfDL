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
	"errors"
	"strings"

	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/service"

	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

func (s *lcmService) CreateEventEndpoint(ctx context.Context, req *service.CreateEventEndpointRequest) (*service.CreateEventEndpointResponse, error) {
	logr := logger.LocLogger(s.logWithCreateEventEndpointRequest(req))
	logr.Infof("Registering event endpoint for job: %s", req.TrainingId)

	if !isValidEventType(req.EventType) {
		logr.Debugf("event type '%s' is invalid", req.EventType)
		return nil, errors.New("invalid event type: " + req.EventType)
	}

	path := req.TrainingId + "/events"

	urlPath := path + "/" + req.EventType + "/" + req.EndpointId

	success, err := s.etcdClient.PutIfKeyMissing(urlPath, req.EndpointUrl, logr)

	if err != nil {
		logr.WithError(err).Errorf("Failed to create endpoint.")
		return nil, err
	}

	if success == false {
		logr.Debugf("Endpoint " + req.EndpointId + " of type " + req.EventType + " already exists.")
		return nil, errors.New("Endpoint already exists")
	}

	return &service.CreateEventEndpointResponse{}, nil
}

func (s *lcmService) DeleteEventEndpoint(ctx context.Context, req *service.DeleteEventEndpointRequest) (*service.DeleteEventEndpointResponse, error) {
	logr := logger.LocLogger(s.logWithDeleteEventEndpointRequest(req))
	logr.Infof("Deleting event endpoint '%s/%s' for job: %s", req.EventType, req.EndpointId, req.TrainingId)

	path := req.TrainingId + "/events/" + req.EventType + "/" + req.EndpointId

	success, err := s.etcdClient.DeleteKeyIfExists(path, logr)

	if err != nil {
		logr.Debugf("deleting zookeeper path '%s' failed: %s", path, err)
		return nil, err
	}
	if success == false {
		logr.Debugf("endpoint " + req.EndpointId + " of type " + req.EventType + " does not exist.");
		return nil, errors.New("Endpoint doesn't exist")
	}
	return &service.DeleteEventEndpointResponse{}, nil
}

func (s *lcmService) GetEventTypeEndpoints(ctx context.Context, req *service.GetEventTypeEndpointsRequest) (*service.GetEventTypeEndpointsResponse, error) {
	logr := logger.LocLogger(s.logWithGetEventTypeEndpointsRequest(req))

	typePath := req.TrainingId + "/events/" + req.EventType + "/"

	rawEndpoints, err := s.etcdClient.Get(typePath, logr, clientv3.WithPrefix())

	if err != nil {
		logr.WithError(err).Errorf("Failed to get endpoints for event type " + req.EventType)
		return nil, errors.New("error finding endpoints for event type: " + req.EventType)
	}

	endpoints := make([]*service.Endpoint, 0, len(rawEndpoints))

	for _, e := range rawEndpoints {
		if e.Value == "" {
			continue
		}
		endpoints = append(endpoints, &service.Endpoint{Id: getIDFromPath(e.Key), Url: e.Value})
	}

	return &service.GetEventTypeEndpointsResponse{Endpoints: endpoints}, nil
}

func (s *lcmService) GetEventEndpoint(ctx context.Context, req *service.GetEventEndpointRequest) (*service.GetEventEndpointResponse, error) {
	logr := logger.LocLogger(s.logWithGetEventEndpointRequest(req))
	logr.Infof("Getting endpoint '%s' for job '%s' of type '%s'", req.EndpointId, req.TrainingId, req.EventType)

	path := req.TrainingId + "/events/" + req.EventType + "/" + req.EndpointId

	endpoint, err := s.etcdClient.Get(path, logr)

	if err != nil {
		logr.WithError(err).Errorf("Failed to get endpoint for event %s of event type %s", req.EndpointId, req.EventType)
		return nil, errors.New("Failed to get endpoint for event " + req.EndpointId + " of event type " + req.EventType)
	}

	url := endpoint[0].Value

	return &service.GetEventEndpointResponse{Url: url, EndpointType: "",}, nil
}

func isValidEventType(eventType string) bool{
	// event types always converted to lowercase for consistency
	lowercase := strings.ToLower(eventType)

	if lowercase == "all"{
		return true
	}
	if lowercase == "status" {
		return true
	}
	if lowercase == "metrics"{
		return true
	}

	return false
}

func getIDFromPath(etcdPath string) string {
	slice := strings.Split(etcdPath, "/")

	return slice[len(slice)-1]
}
