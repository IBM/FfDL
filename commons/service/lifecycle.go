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

package service

import (
	"fmt"
	"net"

	"github.com/IBM/FfDL/commons/config"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// LifecycleHandler provides basic lifecycle methods that each microservice has
// to implement.
type LifecycleHandler interface {
	Start(port int, background bool)
	Stop()
	GetListenerAddress() string
}

// Lifecycle implements the lifecycle operations for microservice including
// dynamic service registration.
type Lifecycle struct {
	Listener        net.Listener
	Server          *grpc.Server
	RegisterService func()
}

type Config struct {
	port       int
	background bool
	tls        bool
	certFile   string
	keyFile    string
}

// Start will start a gRPC microservice on a given port and run it either in
// foreground or background.
func (s *Lifecycle) Start(port int, background bool) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Infof("Starting service at %s", lis.Addr())

	var opts []grpc.ServerOption
	if config.IsTLSEnabled() {
		config.FatalOnAbsentKey(config.ServerCertKey)
		config.FatalOnAbsentKey(config.ServerPrivateKey)

		creds, err := credentials.NewServerTLSFromFile(config.GetServerCert(), config.GetServerPrivateKey())
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	s.Listener = lis
	s.Server = grpc.NewServer(opts...)

	s.RegisterService()
	grpc_health_v1.RegisterHealthServer(s.Server, health.NewServer())

	if background {
		log.Info("ruuning server in background")
		go s.Server.Serve(lis)
	} else {
		log.Info("running server in foreground")
		s.Server.Serve(lis)
	}
}

// Stop will stop the gRPC microservice and the socket.
func (s *Lifecycle) Stop() {
	if s.Listener != nil {
		log.Infof("Stopping service at %s", s.Listener.Addr())
	}
	if s.Server != nil {
		s.Server.GracefulStop()
	}
}

// GetListenerAddress will get the address and port the service is listening.
// Returns the empty string if the service is not running but the method is invoked.
func (s *Lifecycle) GetListenerAddress() string {
	if s.Listener != nil {
		return s.Listener.Addr().String()
	}
	return ""
}
