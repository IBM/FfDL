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
 
 package server

import (
	"crypto/tls"
	"net/http"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/tylerb/graceful"
	log "github.com/sirupsen/logrus"
  mw "github.com/IBM/FfDL/restapi/middleware"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/models"
	"github.com/dre1080/recover"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/training_data"
)

// This file is safe to edit. Once it exists it will not be overwritten

//go:generate swagger generate server --target ../api_v1 --name Dlaas --spec ../api_v1/swagger/swagger.yml --model-package restmodels --server-package server --exclude-main --with-context

func configureFlags(api *operations.DlaasAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.DlaasAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// s.api.Logger = log.Printf

	api.JSONConsumer = runtime.JSONConsumer()

	api.MultipartformConsumer = runtime.DiscardConsumer

	api.JSONProducer = runtime.JSONProducer()

	api.BinProducer = runtime.ByteStreamProducer()

	// Applies when the Authorization header is set with the Basic scheme
	//api.BasicAuthAuth = func(user string, pass string) (interface{}, error) {
	//	// We can assume that the basic authentication is handled by the frontend
	//	// gateway. We don't need any checks here anymore.
	//	if user == "" || pass == "" {
	//		return nil, errors.Unauthenticated("user/password missing")
	//	}
	//	return user, nil
	//}

	// Important: we don't really autenticate here anymore. It is done at the gateway level.
	api.WatsonAuthTokenAuth = func(token string) (interface{}, error) {
		if token == "" {
			return nil, errors.Unauthenticated("token")
		}
		return &service.User{}, nil
	}

	api.WatsonAuthTokenQueryAuth = func(token string) (interface{}, error) {
		return api.WatsonAuthTokenAuth(token)
	}

	// Important: we don't really autenticate here anymore. It is done at the gateway level.
	api.BasicAuthTokenAuth = func(token string) (interface{}, error) {
		return api.WatsonAuthTokenAuth(token)
	}

	api.ModelsDeleteModelHandler = models.DeleteModelHandlerFunc(func(params models.DeleteModelParams, principal interface{}) middleware.Responder {
		return deleteModel(params)
	})
	api.ModelsDownloadModelDefinitionHandler = models.DownloadModelDefinitionHandlerFunc(func(params models.DownloadModelDefinitionParams, principal interface{}) middleware.Responder {
		return downloadModelDefinition(params)
	})
	api.ModelsDownloadTrainedModelHandler = models.DownloadTrainedModelHandlerFunc(func(params models.DownloadTrainedModelParams, principal interface{}) middleware.Responder {
		return downloadTrainedModel(params)
	})
	api.ModelsGetLogsHandler = models.GetLogsHandlerFunc(func(params models.GetLogsParams) middleware.Responder {
		return getLogs(params)
	})
	api.ModelsGetMetricsHandler = models.GetMetricsHandlerFunc(func(params models.GetMetricsParams) middleware.Responder {
		return getMetrics(params)
	})
	api.ModelsGetModelHandler = models.GetModelHandlerFunc(func(params models.GetModelParams, principal interface{}) middleware.Responder {
		return getModel(params)
	})
	api.ModelsListModelsHandler = models.ListModelsHandlerFunc(func(params models.ListModelsParams, principal interface{}) middleware.Responder {
		return listModels(params)
	})
	api.ModelsPostModelHandler = models.PostModelHandlerFunc(func(params models.PostModelParams, principal interface{}) middleware.Responder {
		return postModel(params)
	})
	api.ModelsPatchModelHandler = models.PatchModelHandlerFunc(func(params models.PatchModelParams, principal interface{}) middleware.Responder {
		return patchModel(params)
	})
	api.TrainingDataGetEMetricsHandler = training_data.GetEMetricsHandlerFunc(func(params training_data.GetEMetricsParams, principal interface{}) middleware.Responder {
		return getEMetrics(params)
	})
	api.TrainingDataGetLoglinesHandler = training_data.GetLoglinesHandlerFunc(func(params training_data.GetLoglinesParams, principal interface{}) middleware.Responder {
		return getLoglines(params)
	})

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *graceful.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	logging := mw.NewLoggingMiddleware("rest-api")

	recovery := recover.New(&recover.Options{
		Log: log.Print,
	})

	auth := mw.NewAuthMiddleware(&mw.AuthOptions{
		ExcludedURLs: []string{},
	})

	return logging.Handle(recovery(auth(handler)))
}
