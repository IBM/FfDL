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
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var host string
var port int
var timeout time.Duration
var service string
var debug bool
var tls bool
var caCert string
var serverName string

var rootCmd = &cobra.Command{
	Use:   "grpc-health-checker",
	Short: "gRPC health check utility",
	Long:  "",
	Run:   checkHealth,
}

func main() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&host, "host", "localhost", "gRPC service host")
	flags.IntVarP(&port, "port", "p", 5000, "gRPC service port")
	flags.DurationVarP(&timeout, "timeout", "t", 5000*time.Millisecond, "timeout for the health check")
	flags.StringVarP(&service, "service", "s", "global", "gRPC service name")
	flags.BoolVar(&debug, "debug", false, "turn on debug log")
	flags.BoolVar(&tls, "tls", false, "enable tls")
	flags.StringVar(&caCert, "cacert", "/etc/ssl/dlaas/ca.crt", "CA cert file")
	flags.StringVar(&serverName, "caname", "dlaas.ibm.com", "CA cert server name")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkHealth(cmd *cobra.Command, args []string) {
	if service == "global" { // check the global service
		service = ""
	}
	if debug {
		log.Printf("host arg: %v", host)
		log.Printf("port arg: %v", port)
		log.Printf("timeout arg: %v", timeout)
		log.Printf("service arg: %v", service)
		log.Printf("tls arg: %v", tls)
		log.Printf("cacert arg: %v", caCert)
		log.Printf("sever name arg: %v", serverName)
		log.Printf("debug arg: %v", debug)
	}

	var conn *grpc.ClientConn
	var err error
	if tls {
		creds, err2 := credentials.NewClientTLSFromFile(caCert, serverName)
		if err2 != nil {
			logDebug("Error reading TLS credentials %s\n", err2)
			os.Exit(1)
		}
		conn, err = grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithBlock(), grpc.WithTimeout(timeout), grpc.WithTransportCredentials(creds))
	} else {
		conn, err = grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithBlock(), grpc.WithTimeout(timeout), grpc.WithInsecure())
	}
	if err != nil {
		logDebug("Cannot connect: %s\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	client := grpc_health_v1.NewHealthClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Millisecond)
	defer cancel()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: service,
	})
	if err != nil {
		logDebug("Error: ", err.Error())
		os.Exit(1)
	}
	if resp.Status == grpc_health_v1.HealthCheckResponse_UNKNOWN {
		logDebug("Status UNKNOWN")
		os.Exit(2)
	}
	if resp.Status == grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		logDebug("Status NOT_SERVING")
		os.Exit(3)
	}
}

func logDebug(v ...interface{}) {
	if debug {
		log.Println(v)
	}
}
