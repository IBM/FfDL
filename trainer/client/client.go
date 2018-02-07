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

	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"github.ibm.com/ffdl/ffdl-core/commons/util"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
)

const (
	disabled = "disabled"
	// TrainerV2LocalAddress exposes the local address that is used if we run with DNS disabled
	TrainerV2LocalAddress = ":30005"
)

// TrainerClient is a client interface for interacting with the trainer service.
type TrainerClient interface {
	Client() grpc_trainer_v2.TrainerClient
	Close() error
}

type trainerClient struct {
	client grpc_trainer_v2.TrainerClient
	conn   *grpc.ClientConn
}

// NewTrainer create a new load-balanced client to talk to the Trainer
// service. If the dns_server config option is set to 'disabled', it will
// default to the pre-defined LocalPort of the service.
func NewTrainer() (TrainerClient, error) {
	return NewTrainerWithAddress(TrainerV2LocalAddress)
}

// NewTrainerWithAddress create a new load-balanced client to talk to the Trainer
// service. If the dns_server config option is set to 'disabled', it will
// default to the pre-defined LocalPort of the service.
func NewTrainerWithAddress(addr string) (TrainerClient, error) {
	address := fmt.Sprintf("ffdl-trainer.%s.svc.cluster.local:80", config.GetPodNamespace())
	dnsServer := viper.GetString("dns_server")
	if dnsServer == disabled { // for local testing without DNS server
		address = addr
	}

	dialOpts, err := util.CreateClientDialOpts()
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		log.Errorf("Could not connect to trainer service: %v", err)
		return nil, err
	}

	return &trainerClient{
		conn:   conn,
		client: grpc_trainer_v2.NewTrainerClient(conn),
	}, nil
}

func (c *trainerClient) Client() grpc_trainer_v2.TrainerClient {
	return c.client
}

func (c *trainerClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
