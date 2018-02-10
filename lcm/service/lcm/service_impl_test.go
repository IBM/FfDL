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
	"testing"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/lcm/lcmconfig"

	"github.com/IBM/FfDL/commons/service"

	"github.com/IBM/FfDL/commons/logger"
	v1core "k8s.io/api/core/v1"
	// "github.com/coreos/etcd/clientv3"
	// "github.com/IBM/FfDL/lcm/coord"
)

func init() {
	config.InitViper()
}

func TestCalcMemory(t *testing.T) {
	r := &service.ResourceRequirements{
		Memory:     512,
		MemoryUnit: service.ResourceRequirements_MB,
	}
	assert.EqualValues(t, 512, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     8.5,
		MemoryUnit: service.ResourceRequirements_GB,
	}
	assert.EqualValues(t, 8500, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     128,
		MemoryUnit: service.ResourceRequirements_MiB,
	}
	assert.EqualValues(t, 134.22, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     8.2,
		MemoryUnit: service.ResourceRequirements_GiB,
	}
	assert.EqualValues(t, 8804.68, calcMemory(r))

}

func DISABLETestDefineLearnerJobs(t *testing.T) {
	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

	s, err := newService()
	assert.Nil(t, err)
	assert.NotNil(t, s)
	//spew.Dump(s)
	fmt.Printf("s: %+v\n", s)

	// Create a training job.
	name := "single-node-integration-test"
	req := &service.JobDeploymentRequest{
		Name:       name,
		UserId:     name,
		TrainingId: name,
		Framework:  "caffe",
		Version:    "1",
		Resources:  &service.ResourceRequirements{Gpus: 1},
	}
	envs := []v1core.EnvVar{
		v1core.EnvVar{Name: "DATA_DIR", Value: "/dl-data"},
		v1core.EnvVar{Name: "DATA_STORE_APIKEY", Value: "e739397fbd7d3603679a87a8dcf7b7e66b8892fbcbc233e1d8e8932efd6eed84"},
		v1core.EnvVar{Name: "DATA_STORE_AUTHURL", Value: "https://dal05.objectstorage.service.networklayer.com/auth/v1.0/"},
		v1core.EnvVar{Name: "DATA_STORE_OBJECTID", Value: "mnist_lmdb_data"},
		v1core.EnvVar{Name: "DATA_STORE_USERNAME", Value: "IBMOS366226-358:dlaas-watson"},
		v1core.EnvVar{Name: "MODEL_DIR", Value: "/model-code"},
		v1core.EnvVar{Name: "MODEL_STORE_APIKEY", Value: "TPqhRQQGaYW5Ew3qpVT4CdYJhJ1lJgpIYhJPY5Yt"},
		v1core.EnvVar{Name: "MODEL_STORE_AUTHURL", Value: "https://s3-api.us-geo.objectstorage.service.networklayer.com"},
		v1core.EnvVar{Name: "MODEL_STORE_OBJECTID", Value: "dlaas-models/caffe-model-zy9uxrGzR.zip"},
		v1core.EnvVar{Name: "MODEL_STORE_USERNAME", Value: "idd4n47KjaNrJ891JE6S"},
		v1core.EnvVar{Name: "RESULT_DIR", Value: "/dl-models"},
		v1core.EnvVar{Name: "RESULT_STORE_APIKEY", Value: "e739397fbd7d3603679a87a8dcf7b7e66b8892fbcbc233e1d8e8932efd6eed84"},
		v1core.EnvVar{Name: "RESULT_STORE_AUTHURL", Value: "https://dal05.objectstorage.service.networklayer.com/auth/v1.0/"},
		v1core.EnvVar{Name: "RESULT_STORE_OBJECTID", Value: "mnist_trained_model/training-lcm-integration-test"},
		v1core.EnvVar{Name: "RESULT_STORE_USERNAME", Value: "IBMOS366226-358:dlaas-watson"},
		v1core.EnvVar{Name: "TRAINING_COMMAND", Value: "caffe train -solver lenet_solver.prototxt"},
	}

	deploySpec, err := defineLearnerDeployment(s, req, 1, "dev_v8", envs, 0, logr)
	assert.Nil(t, err)
	assert.NotNil(t, deploySpec)

	s.killDeployedJob(name, name, name)
	defer s.killDeployedJob(name, name, name)

	_, err = s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).Create(deploySpec)
	assert.Nil(t, err)

	logr.Printf("Created training job %s", name)
	//time.Sleep(8 * time.Second)

}

func TestIsValidEventType(t *testing.T) {
	// valid event types are status, metrics, logs, and all.
	// (all user inputs converted to lowercase)

	// standard inputs
	assert.EqualValues(t, true, isValidEventType("all"))
	assert.EqualValues(t, true, isValidEventType("status"))
	assert.EqualValues(t, true, isValidEventType("metrics"))

	// strange caps
	assert.EqualValues(t, true, isValidEventType("All"))
	assert.EqualValues(t, true, isValidEventType("METRICS"))
	assert.EqualValues(t, true, isValidEventType("sTaTuS"))

	// bad inputs
	assert.EqualValues(t, false, isValidEventType("al"))
	assert.EqualValues(t, false, isValidEventType("hello"))
	assert.EqualValues(t, false, isValidEventType("??!!.:)"))
	assert.EqualValues(t, false, isValidEventType("  "))
}

/*
// 	WILL FIX IN A LATER PR
// 		waiting for NewTestService in service_impl.go

// maybe also write a test where creating an endpoint is supposed to fail
// i.e. invalid event type

func TestCreateEventEndpoint(t *testing.T) {
		trainingId := "test-id"
		eventType := "all"
		endpointId := "test-endpoint3"
		endpointUrl := "abc.def/xyz"

		req := &service.CreateEventEndpointRequest{
			TrainingId: 	trainingId,
			UserId: 			"test-user", // unnecessary at the moment
			EventType: 		eventType,
			EndpointUrl: 	endpointUrl,
			EndpointType: "url", // unnecessary at the moment
			EndpointId: 	endpointId,
		}
		path := trainingId + "/events/" + eventType + "/" + endpointId

		logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

		lcm := &lcmService{} //newService() //&lcmService{} //NewTestService()

		coordinator, err := coord.NewCoordinator(coord.Config{Endpoints: config.GetEtcdEndpoints(), Prefix: "",
			Cert: config.GetEtcdCertLocation(), Username: config.GetEtcdUsername(), Password: config.GetEtcdPassword()}, logr)

		if err != nil {
			fmt.Println(err.Error)
		}

		lcm.etcdClient = coordinator

		// cleanup later
		// defer zk.DeleteRecursive(trainingId)
		defer lcm.etcdClient.DeleteKeyWithOpts(trainingId, logr, clientv3.WithPrefix())

		_, err = lcm.CreateEventEndpoint(nil, req)
		assert.EqualValues(t, nil, err)

		value, err := lcm.etcdClient.Get(path, logr)
		assert.Nil(t, err)
		assert.EqualValues(t, endpointUrl, value)

		// should fail when trying to create the same endpointUrl
		_, err = lcm.CreateEventEndpoint(nil, req)
		assert.NotEqual(t, nil, err)


		// cleanup
		dreq := &service.DeleteEventEndpointRequest{
			TrainingId: 	trainingId,
			UserId: 			"test-user", // unnecessary at the moment
			EventType: 		eventType,
			EndpointId: 	endpointId,
		}
		lcm.DeleteEventEndpoint(nil, dreq)

}


func TestDeleteEventEndpoint(t *testing.T) {
	trainingId := "test-id1"
	eventType := "all"
	endpointId := "test-endpoint2"
	endpointUrl := "abc.def/xyz"

	req := &service.DeleteEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointId: 	endpointId,
	}

	creq := &service.CreateEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointUrl: 	endpointUrl,
		EndpointType: "url", // unnecessary at the moment
		EndpointId: 	endpointId,
	}

	path := trainingId + "/events/" + eventType + "/" + endpointId

	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

	lcm := &lcmService{} //NewTestService()

	// cleanup later
	// defer zk.DeleteRecursive(trainingId)
	defer lcm.etcdClient.DeleteKeyWithOpts(trainingId, logr, clientv3.WithPrefix())

	_, err := lcm.DeleteEventEndpoint(nil, req)
	assert.NotEqual(t, nil, err)

	_, err = lcm.CreateEventEndpoint(nil, creq)
	assert.EqualValues(t, nil, err)

	_, err = lcm.DeleteEventEndpoint(nil, req)
	assert.EqualValues(t, nil, err)

// Doublecheck
	value, _ := lcm.etcdClient.Get(path, logr)
	assert.EqualValues(t, nil, value)

}


// get event type endpoints

func TestGetEventTypeEndpoints(t *testing.T) {
	trainingId := "test-id2"
	eventType := "all"
	endpointUrl := "abc.def/xyz"

	req := &service.GetEventTypeEndpointsRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
	}

	// path := trainingId + "/events/" + eventType + "/1"

	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

	lcm := &lcmService{}

	// cleanup later
	// defer zk.DeleteRecursive(trainingId)
	defer lcm.etcdClient.DeleteKeyWithOpts(trainingId, logr, clientv3.WithPrefix())

	// no events yet
	_, err := lcm.GetEventTypeEndpoints(nil, req)
	assert.NotEqual(t, nil, err)

	creq := &service.CreateEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointUrl: 	endpointUrl,
		EndpointType: "url", // unnecessary at the moment
		EndpointId: 	"1",
	}
	_, err = lcm.CreateEventEndpoint(nil, creq)
	assert.EqualValues(t, nil, err)

	// should be 1 event of type "all"
	res, err := lcm.GetEventTypeEndpoints(nil, req)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 1, len(res.Endpoints))


	creq = &service.CreateEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointUrl: 	endpointUrl,
		EndpointType: "slack", // unnecessary at the moment
		EndpointId: 	"2",
	}
	_, err = lcm.CreateEventEndpoint(nil, creq)
	assert.EqualValues(t, nil, err)

	req = &service.GetEventTypeEndpointsRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
	}

	res, err = lcm.GetEventTypeEndpoints(nil, req)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, 2, len(res.Endpoints))

	// different event type should not have any events yet
	req = &service.GetEventTypeEndpointsRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		"logs",
	}
	_, err = lcm.GetEventTypeEndpoints(nil, req)
	assert.NotEqual(t, nil, err)

}

// getEventEndpoint
func TestGetEventEndpoint(t *testing.T) {
	trainingId := "test-id3"
	eventType := "all"
	endpointUrl := "abc.def/xyz"

	req := &service.GetEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointId: 	"1",
	}

	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

	lcm := &lcmService{}

	// cleanup later
	// defer zk.DeleteRecursive(trainingId)
	defer lcm.etcdClient.DeleteKeyWithOpts(trainingId, logr, clientv3.WithPrefix())

	// no events yet
	_, err := lcm.GetEventEndpoint(nil, req)
	assert.NotEqual(t, nil, err)

	creq := &service.CreateEventEndpointRequest{
		TrainingId: 	trainingId,
		UserId: 			"test-user", // unnecessary at the moment
		EventType: 		eventType,
		EndpointUrl: 	endpointUrl,
		EndpointType: "url", // unnecessary at the moment
		EndpointId: 	"1",
	}
	_, err = lcm.CreateEventEndpoint(nil, creq)
	assert.EqualValues(t, nil, err)

	// should be 1 event of type "all"
	res, err := lcm.GetEventEndpoint(nil, req)
	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, "abc.def/xyz", res.Url)

}


// get metrics
func TestGetMetrics(t *testing.T) {
	trainingId := "test-4"

	// metricsPath := trainingId + "/" + zkLearners + "/" + zkLearner + "1" + "_" + zkStatus + "_summary_metrics"
	metricsPath := fmt.Sprintf("%s/learners/learner_1/%s", trainingId, "summary_metrics")

	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))

	lcm := &lcmService{}

	// cleanup later
	// defer zk.DeleteRecursive(trainingId)
	defer lcm.etcdClient.DeleteKeyWithOpts(trainingId, logr, clientv3.WithPrefix())

	// zk.Create(metricsPath, []byte("Hello world"), logr.Logger)
	lcm.etcdClient.Put(metricsPath, "Hello world", logr)

	req := &service.GetMetricsRequest{
		TrainingId: 	trainingId,
	}

	res, err := lcm.GetMetrics(nil, req)

	assert.EqualValues(t, nil, err)
	assert.EqualValues(t, "Hello world", res.Metrics)

}
*/
