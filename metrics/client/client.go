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
	"fmt"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/IBM/FfDL/metrics/service/grpc_training_data_v1"
	"google.golang.org/grpc"

	"github.com/IBM/FfDL/commons/logger"
	// "github.com/IBM/FfDL/metrics/service"
)

const (
	disabled = "disabled"

	// TDSLocalAddress exposes the local address that is used if we run with DNS disabled
	TDSLocalAddress = ":30015"
)

// TrainingDataClient is a client interface for interacting with the training metrics service.
type TrainingDataClient interface {
	Client() grpc_training_data_v1.TrainingDataClient
	Close() error
}

type trainerStatusClient struct {
	client grpc_training_data_v1.TrainingDataClient
	conn   *grpc.ClientConn
}

// NewTrainingDataClient create a new load-balanced client to talk to the training metrics
// service. If the dns_server config option is set to 'disabled', it will
// default to the pre-defined LocalPort of the service.
func NewTrainingDataClient() (TrainingDataClient, error) {
	return NewTrainingDataClientWithAddress(TDSLocalAddress)
}

// NewTrainingDataClientFromExisting creates a wrapper around an existing client.  Used at least for mock clients.
//noinspection GoUnusedExportedFunction
func NewTrainingDataClientFromExisting(tds grpc_training_data_v1.TrainingDataClient ) (TrainingDataClient, error) {
	return &trainerStatusClient{
		conn:   nil,
		client: tds,
	}, nil
}


// NewTrainingDataClientWithAddress create a new load-balanced client to talk to the training metrics
// service. If the dns_server config option is set to 'disabled', it will
// default to the pre-defined LocalPort of the service.
func NewTrainingDataClientWithAddress(addr string) (TrainingDataClient, error) {
	logr := logger.LocLogger(logger.LogServiceBasic("training-data-service"))
	logr.Debugf("function entry")

	var address string
	dnsServer := viper.GetString("dns_server")
	if dnsServer == disabled { // for local testing without DNS server
		address = addr
		logr.Debugf("DNS disabled: Running passed in address: %v", address)
	} else {
		address = fmt.Sprintf("ffdl-trainingdata.%s.svc.cluster.local:80", config.GetPodNamespace())
		logr.Debugf("ffdl-trainingdata address: %v", address)
	}
	logr.Debugf("IsTLSEnabled: %t", config.IsTLSEnabled())
	logr.Debugf("final address: %v", address)

	dialOpts, err := util.CreateClientDialOpts()
	if err != nil {
		return nil, err
	}
	for i, v := range dialOpts {
		log.Printf("dialOpts[%d]: %+v", i, v)
	}

	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		log.Errorf("Could not connect to trainer service: %v", err)
		return nil, err
	}

	logr.Debugf("function exit")

	return &trainerStatusClient{
		conn:   conn,
		client: grpc_training_data_v1.NewTrainingDataClient(conn),
	}, nil
}

func (c *trainerStatusClient) Client() grpc_training_data_v1.TrainingDataClient {
	return c.client
}

func (c *trainerStatusClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
