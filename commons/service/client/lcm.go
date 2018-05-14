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

	"github.com/grpc-ecosystem/go-grpc-prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/commons/util"
	"google.golang.org/grpc"
)

const (
	disabled = "disabled"
	// LcmLocalPort exposes the port that is used if we run with DNS disabled
	LcmLocalPort = 30002
	// LcmLocalAddress exposes the local address that is used if we run with DNS disabled
	LcmLocalAddress = ":30002"
)

// LcmClient is a client interface for interacting with the LCM service.
type LcmClient interface {
	Client() service.LifecycleManagerClient
	Close() error
}

type lcmClient struct {
	client service.LifecycleManagerClient
	conn   *grpc.ClientConn
}

// NewLcm create a new client to talk to the LifecycleManager
// service. If the dns_server config option is set to 'disabled/, it will
// default to localhost:port.
func NewLcm(lcm service.LifecycleManagerClient) (LcmClient, error) {
	address := fmt.Sprintf("ffdl-lcm.%s.svc.cluster.local:80", config.GetPodNamespace())
	dnsServer := viper.GetString("dns_server")
	if dnsServer == disabled { // for local testing without DNS server
		address = LcmLocalAddress
	}

	dialOpts, err := util.CreateClientDialOpts()
	if err != nil {
		return nil, err
	}

	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor), grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor))

	if lcm == nil {
		conn, err := grpc.Dial(address, dialOpts...)
		if err != nil {
			log.Errorf("Could not connect to lcm service: %v", err)
			return nil, err
		}
		return &lcmClient{
			conn:   conn,
			client: service.NewLifecycleManagerClient(conn),
		}, nil
	}

	return &lcmClient{
		conn:   nil,
		client: lcm,
	}, nil
}

func (c *lcmClient) Client() service.LifecycleManagerClient {
	return c.client
}

func (c *lcmClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
