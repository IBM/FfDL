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
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tylerb/graceful"

	"github.com/go-openapi/loads"

	"github.com/IBM/FfDL/restapi/api_v1/server"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations"
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
)

func main() {
	config.InitViper()
	logger.Config()

	swaggerSpec, err := loads.Analyzed(server.SwaggerJSON, "")
	if err != nil {
		log.Fatalln(err)
	}

	api := operations.NewDlaasAPI(swaggerSpec)
	// api.Logger = log.Printf

	srv := server.NewServer(api)
	defer srv.Shutdown()
	srv.Port = viper.GetInt(config.PortKey)
	srv.ConfigureAPI()

	mux := http.NewServeMux()
	mux.Handle("/", srv.GetHandler())
	mux.HandleFunc("/health", server.GetHealth)

	address := fmt.Sprintf(":%d", srv.Port)
	log.Printf("DLaaS REST API v1 serving on %s", address)
	err = graceful.RunWithErr(address, 10*time.Second, mux)
	if err != nil {
		log.Fatalln(err)
	}
}
