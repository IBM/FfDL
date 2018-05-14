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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	mw "github.com/IBM/FfDL/restapi/middleware"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/IBM/FfDL/restapi/api_v1/restmodels"
	"github.com/IBM/FfDL/restapi/api_v1/server/operations/models"
	trainerClient "github.com/IBM/FfDL/trainer/client"

	"bufio"
	"bytes"
	"io"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
	"golang.org/x/net/websocket"

	"time"

	"github.com/IBM/FfDL/restapi/api_v1/server/operations/training_data"
	trainingDataClient "github.com/IBM/FfDL/metrics/client"

	"github.com/IBM/FfDL/metrics/service/grpc_training_data_v1"
)

const (
	defaultLogPageSize = 10
)

// postModel posts a model definition and starts the training
func postModel(params models.PostModelParams) middleware.Responder {
	logr := logger.LocLogger(logWithPostModelParams(params))
	logr.Debugf("postModel invoked: %v", params.HTTPRequest.Header)

	manifestBytes, err := ioutil.ReadAll(params.Manifest.Data)
	if err != nil {
		msg := fmt.Sprintf("Cannot read 'manifest' parameter: %s", err.Error())
		logr.Errorf(msg)
		return models.NewPostModelBadRequest().WithPayload(&restmodels.Error{
			Code:        400,
			Error:       msg,
			Description: "Incorrect parameters",
		})
	}
	logr.Debug("Loading Manifest")
	manifest, err := LoadManifestV1(manifestBytes)
	if err != nil {
		logr.WithError(err).Errorf("Parameter 'manifest' contains incorrect YAML")
		return models.NewPostModelBadRequest().WithPayload(&restmodels.Error{
			Code:        400,
			Description: "Incorrect manifest",
			Error:       err.Error(),
		})
	}

	modelDefinition, err := ioutil.ReadAll(params.ModelDefinition.Data)
	if err != nil {
		logr.Errorf("Cannot read 'model_definition' parameter: %s", err.Error())
		return models.NewPostModelBadRequest().WithPayload(&restmodels.Error{
			Code:        400,
			Description: "Incorrect parameters",
			Error:       err.Error(),
		})
	}

	// TODO this is an interim HACK b/c with the new rest API in combination with the OLD
	// manifest we do not have a way to select the from a list of dataStores so we cap it to only one.
	if len(manifest.DataStores) > 1 {
		return models.NewPostModelBadRequest().WithPayload(&restmodels.Error{
			Code:        400,
			Description: "",
			Error:       "Please only specify one data_store in the manifest. This constraint will go away with the once the new manifest is in place.",
		})
	}

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainer.Close()

	// TODO do some basic manifest.yml validation to avoid a panic

	tresp, err := trainer.Client().CreateTrainingJob(params.HTTPRequest.Context(),
		manifest2TrainingRequest(manifest, modelDefinition, params.HTTPRequest, logr))

	if err != nil {
		logr.WithError(err).Errorf("Trainer service call failed")
		if grpc.Code(err) == codes.InvalidArgument || grpc.Code(err) == codes.NotFound {
			return models.NewPostModelBadRequest().WithPayload(
				&restmodels.Error{
					Description: "",
					Error:       grpc.ErrorDesc(err),
					Code:        400,
				})
		}

		if grpc.Code(err) == codes.ResourceExhausted {
			return models.NewPostModelBadRequest().WithPayload(&restmodels.Error{
				Code:        http.StatusTooManyRequests,
				Description: grpc.ErrorDesc(err),
				Error:       grpc.ErrorDesc(err),
			})
		}

		return error500(logr, "")
	}

	loc := params.HTTPRequest.URL.Path + "/" + tresp.TrainingId
	return models.NewPostModelCreated().
		WithLocation(loc).
		WithPayload(&restmodels.BasicNewModel{
			BasicModel: restmodels.BasicModel{
				ModelID: tresp.TrainingId,
			},
			Location: loc,
		})
}

func deleteModel(params models.DeleteModelParams) middleware.Responder {
	logr := logger.LocLogger(logWithDeleteModelParams(params))
	logr.Debugf("deleteModel invoked: %v", params.HTTPRequest.Header)

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainer.Close()

	_, err = trainer.Client().DeleteTrainingJob(params.HTTPRequest.Context(), &grpc_trainer_v2.DeleteRequest{
		TrainingId: params.ModelID,
		UserId:     getUserID(params.HTTPRequest),
	})
	if err != nil {
		logr.WithError(err).Errorf("Trainer GetTrainingJob service call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewDeleteModelUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		if grpc.Code(err) == codes.NotFound {
			return models.NewDeleteModelNotFound().WithPayload(&restmodels.Error{
				Error:       "Not found",
				Code:        http.StatusNotFound,
				Description: "",
			})
		}
		return error500(logr, "")
	}
	//logr.Debugf("Trainer GetTrainingJob response: %s", rresp.String())

	return models.NewDeleteModelOK().WithPayload(
		&restmodels.BasicModel{
			ModelID: params.ModelID,
		})
}

func getModel(params models.GetModelParams) middleware.Responder {
	logr := logger.LocLogger(logWithGetModelParams(params))
	logr.Debugf("getModel invoked: %v", params.HTTPRequest.Header)

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainer.Close()

	rresp, err := trainer.Client().GetTrainingJob(params.HTTPRequest.Context(), &grpc_trainer_v2.GetRequest{
		TrainingId: params.ModelID,
		UserId:     getUserID(params.HTTPRequest),
	})
	if err != nil {
		logr.WithError(err).Errorf("Trainer GetTrainingJob service call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewGetModelUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		if grpc.Code(err) == codes.NotFound {
			return models.NewGetModelNotFound().WithPayload(&restmodels.Error{
				Error:       "Not found",
				Code:        http.StatusNotFound,
				Description: "",
			})
		}
		return error500(logr, "")
	}
	//logr.Debugf("Trainer GetTrainingJob response: %s", rresp.String())

	if rresp.Job == nil {
		return models.NewGetModelNotFound().WithPayload(&restmodels.Error{
			Error:       "Not found",
			Code:        http.StatusNotFound,
			Description: "",
		})
	}

	m := createModel(params.HTTPRequest, rresp.Job)

	//logr.Debugf("m: %+v", m)
	return models.NewGetModelOK().WithPayload(m)
}

func listModels(params models.ListModelsParams) middleware.Responder {
	logr := logger.LocLogger(logWithGetListModelsParams(params))

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainer.Close()

	logr.Debugf("Calling trainer.Client().GetAllTrainingsJobs(...)")
	resp, err := trainer.Client().GetAllTrainingsJobs(params.HTTPRequest.Context(), &grpc_trainer_v2.GetAllRequest{
		UserId: getUserID(params.HTTPRequest),
	})

	if err != nil {
		logr.WithError(err).Error("Trainer readAll service call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewListModelsUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		return error500(logr, "")
	}

	// build up response
	marr := make([]*restmodels.Model, 0, len(resp.Jobs))
	logr.Debugf("Number of training jobs found: %d", len(resp.Jobs))
	for _, job := range resp.Jobs {
		// use append(); we may have skipped some because they were gone by the time we got to them.
		marr = append(marr, createModel(params.HTTPRequest, job))
	}
	return models.NewListModelsOK().WithPayload(&restmodels.ModelList{
		Models: marr,
	})
}

func downloadModelDefinition(params models.DownloadModelDefinitionParams) middleware.Responder {
	logr := logger.LocLogger(logWithDownloadModelDefinitionParams(params))
	logr.Debugf("downloadModelDefinition invoked: %v", params.HTTPRequest.Header)

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		log.WithError(err).Error("Cannot create client for trainer service")
		return error500(logr, "")
	}

	stream, err := trainer.Client().GetModelDefinition(params.HTTPRequest.Context(), &grpc_trainer_v2.ModelDefinitionRequest{
		TrainingId: params.ModelID,
		UserId:     getUserID(params.HTTPRequest),
	})
	if err != nil {
		logr.WithError(err).Error("Trainer GetModelDefinition service call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewDownloadModelDefinitionUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		if grpc.Code(err) == codes.NotFound {
			return models.NewDownloadModelDefinitionNotFound().WithPayload(&restmodels.Error{
				Error:       "Not found",
				Code:        http.StatusNotFound,
				Description: "",
			})
		}
		return error500(logr, "")
	}

	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		defer trainer.Close()

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				w.WriteHeader(200)
				if err = stream.CloseSend(); err != nil {
					logr.WithError(err).Error("Closing stream failed")
				}
				break
			} else if err != nil {
				logr.WithError(err).Errorf("Cannot read model definition")
				// this error handling is a bit of a hack with overriding the content-type
				// but the swagger-generated REST client chokes if we leave the content-type
				// as application-octet stream
				w.WriteHeader(500)
				w.Header().Set(runtime.HeaderContentType, runtime.JSONMime)
				payload, _ := json.Marshal(&restmodels.Error{
					Error:       "Internal server error",
					Description: "",
					Code:        500,
				})
				w.Write(payload)
				break
			}
			w.Write(chunk.Data)
		}
	})
}

func downloadTrainedModel(params models.DownloadTrainedModelParams) middleware.Responder {
	logr := logger.LocLogger(logWithDownloadTrainedModelParams(params))
	logr.Debugf("downloadTrainedModel invoked: %v", params.HTTPRequest.Header)

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		log.WithError(err).Error("Cannot create client for trainer service")
		return error500(logr, "")
	}

	stream, err := trainer.Client().GetTrainedModel(params.HTTPRequest.Context(), &grpc_trainer_v2.TrainedModelRequest{
		TrainingId: params.ModelID,
		UserId:     getUserID(params.HTTPRequest),
	})
	if err != nil {
		logr.WithError(err).Errorf("Trainer GetTrainedModel service call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewDownloadTrainedModelUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		if grpc.Code(err) == codes.NotFound {
			return models.NewDownloadTrainedModelNotFound().WithPayload(&restmodels.Error{
				Error:       "Not found",
				Code:        http.StatusNotFound,
				Description: "",
			})
		}
		return error500(logr, "")
	}

	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		defer trainer.Close()

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				w.WriteHeader(200)
				if err = stream.CloseSend(); err != nil {
					logr.WithError(err).Error("Closing stream failed")
				}
				break
			} else if err != nil {
				logr.WithError(err).Error("Cannot read trained model")
				// this error handling is a bit of a hack with overriding the content-type
				// but the swagger-generated REST client chokes if we leave the content-type
				// as application-octet stream
				w.WriteHeader(500)
				w.Header().Set(runtime.HeaderContentType, runtime.JSONMime)
				payload, _ := json.Marshal(&restmodels.Error{
					Error:       "Internal server error",
					Description: "",
					Code:        500,
				})
				w.Write(payload)
				break
			}
			w.Write(chunk.Data)
		}
	})
}

// getTrainingLogs establishes a long running http connection and streams the logs of the training container.
func getLogsOrMetrics(params models.GetLogsParams, isMetrics bool) middleware.Responder {
	logr := logger.LocLogger(logWithGetLogsParams(params))

	ctx := params.HTTPRequest.Context()

	isFollow := (params.HTTPRequest.Header.Get("Sec-Websocket-Key") != "")

	if !isFollow {
		isFollow = *params.Follow
	}

	logr.Debugf("isFollow: %v, isMetrics: %v", isFollow, isMetrics)

	var timeout time.Duration

	if params.ModelID == "wstest" {
		logr.Debug("is wstest")
		timeout = 2 * time.Minute

	} else if isFollow {
		// Make this a *very* long time out.  In the longer run, if we
		// push to a message queue, we can hopefully just subscribe to a web
		// socket.
		timeout = 80 * time.Hour
	} else {
		timeout = 3 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	// don't cancel here as we are passing the cancel function to others.

	// HACK FOR WS TEST
	if params.ModelID == "wstest" {
		return getTrainingLogsWSTest(ctx, params, cancel)
	}

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Error("Cannot create client for lcm service")
		defer cancel()
		return error500(logr, "")
	}

	var stream grpc_trainer_v2.Trainer_GetTrainedModelLogsClient

	if isMetrics {
//		stream, err = trainer.Client().GetTrainedModelMetrics(ctx, &grpc_trainer_v2.TrainedModelMetricsRequest{
//			TrainingId: params.ModelID,
//			UserId:     getUserID(params.HTTPRequest),
//			Follow:     isFollow,
//		})
		logr.WithError(err).Errorf("GetTrainedModelMetrics has been removed")
		defer cancel()
		return error500(logr, "")
	} else {
		stream, err = trainer.Client().GetTrainedModelLogs(ctx, &grpc_trainer_v2.TrainedModelLogRequest{
			TrainingId: params.ModelID,
			UserId:     getUserID(params.HTTPRequest),
			Follow:     isFollow,
		})
	}
	if err != nil {
		logr.WithError(err).Errorf("GetTrainedModelLogs failed")
		defer cancel()
		return error500(logr, "")
	}
	// _, err = *stream.Header()
	// We seem to need this pause, which is somewhat disconcerting. -sb
	//time.Sleep(time.Second * 2)

	if params.HTTPRequest.Header.Get("Sec-Websocket-Key") != "" {
		return getTrainingLogsWS(trainer, params, stream, cancel, isMetrics)
	}

	return middleware.ResponderFunc(func(w http.ResponseWriter, prod runtime.Producer) {
		// The close of the LcmClient should also close the stream, as far as i can tell.
		defer trainer.Close()
		defer cancel()

		//logr.Debugln("w.WriteHeader(200)")
		w.WriteHeader(200)
		// w.Header().Set("Transfer-Encoding", "chunked")
		var onelinebytes []byte
		for {
			var logFrame *grpc_trainer_v2.ByteStreamResponse
			// logr.Debugln("CALLING stream.Recv()")
			logFrame, err := stream.Recv()
			// time.Sleep(time.Second * 2)
			if logFrame == nil {
				if err != io.EOF && err != nil {
					logr.WithError(err).Errorf("stream.Recv() returned error")
				}
				break
			}
			if isMetrics {
				var bytesBuf []byte = logFrame.GetData()
				if bytesBuf != nil {

					byteReader := bytes.NewReader(bytesBuf)
					bufioReader := bufio.NewReader(byteReader)
					for {
						var lineBytes []byte
						lineBytes, err := bufioReader.ReadBytes('\n')
						if lineBytes != nil {
							lenRead := len(lineBytes)

							if err == nil || (err == io.EOF && lenRead > 0 && lineBytes[lenRead-1] == '}') {

								if onelinebytes != nil {
									lineBytes = append(onelinebytes[:], lineBytes[:]...)
									onelinebytes = nil
								}

								_, err := w.Write(lineBytes)
								//logr.Debugf("w.Write(bytes) says %d bytes written", n)
								if err != nil && err != io.EOF {
									logr.Errorf("getTrainingLogs(2) Write returned error: %s", err.Error())
								}
								// logr.Debugln("if f, ok := w.(http.Flusher); ok {")
								if f, ok := w.(http.Flusher); ok {
									logr.Debugln("f.Flush()")
									f.Flush()
								}
							} else {
								if onelinebytes == nil {
									onelinebytes = lineBytes
								} else {
									onelinebytes = append(onelinebytes[:], lineBytes[:]...)
								}
							}
						}
						if err == io.EOF {
							break
						}
					}
				}
			} else {
				var bytes []byte = logFrame.GetData()
				if bytes != nil {
					//logr.Debugln("w.Write(bytes) len = %d", len(bytes))
					_, err := w.Write(bytes)
					//logr.Debugf("w.Write(bytes) says %d bytes written", n)
					if err != nil && err != io.EOF {
						logr.Errorf("getTrainingLogs(2) Write returned error: %s", err.Error())
					}
					//logr.Debugln("if f, ok := w.(http.Flusher); ok {")
					if f, ok := w.(http.Flusher); ok {
						logr.Debugln("f.Flush()")
						f.Flush()
					}
				}
			}
			//logr.Debugln("bottom of for")
		}
	})
}

func getLogs(params models.GetLogsParams) middleware.Responder {
	return getLogsOrMetrics(params, false)
}

func makeRestAnyMapFromGrpcAnyMap(restMap map[string]*grpc_training_data_v1.Any) map[string]restmodels.V1Any {
	grpcMap := make(map[string]restmodels.V1Any)
	for k, v := range restMap {
		var restType restmodels.AnyDataType
		switch v.Type {
		case grpc_training_data_v1.Any_STRING:
			restType = restmodels.AnyDataTypeSTRING
			break
		case grpc_training_data_v1.Any_JSONSTRING:
			restType = restmodels.AnyDataTypeJSONSTRING
			break
		case grpc_training_data_v1.Any_INT:
			restType = restmodels.AnyDataTypeINT
			break
		case grpc_training_data_v1.Any_FLOAT:
			restType = restmodels.AnyDataTypeFLOAT
			break

		}
		grpcMap[k] = restmodels.V1Any{restType, v.Value}
	}
	return grpcMap
}

func makeGrpcSearchTypeFromRestSearchType(st string) grpc_training_data_v1.Query_SearchType {
	var searchType grpc_training_data_v1.Query_SearchType
	switch restmodels.QuerySearchType(st) {
	case restmodels.QuerySearchTypeTERM:
		searchType = grpc_training_data_v1.Query_TERM
		break
	case restmodels.QuerySearchTypeMATCH:
		searchType = grpc_training_data_v1.Query_MATCH
		break
	case restmodels.QuerySearchTypeNESTED:
		searchType = grpc_training_data_v1.Query_NESTED
		break
	case restmodels.QuerySearchTypeALL:
		searchType = grpc_training_data_v1.Query_ALL
		break
	}
	return searchType
}

func getEMetrics(params training_data.GetEMetricsParams) middleware.Responder {
	logr := logger.LocLogger(logWithEMetricsParams(params))
	logr.Debug("function entry")

	trainingData, err := trainingDataClient.NewTrainingDataClient()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainingData.Close()

	var metaUserID = getUserID(params.HTTPRequest)

	var sinceQuery = ""
	if params.SinceTime != nil {
		sinceQuery = *params.SinceTime
	}
	var pagesize int32 = defaultLogPageSize
	if params.Pagesize != nil {
		pagesize = *params.Pagesize
	}
	var pos int64
	if params.Pos != nil {
		pos = *params.Pos
	}
	var searchType grpc_training_data_v1.Query_SearchType = grpc_training_data_v1.Query_TERM
	if params.SearchType != nil {
		searchType = makeGrpcSearchTypeFromRestSearchType(*params.SearchType)
	}

	query := &grpc_training_data_v1.Query{
		Meta: &grpc_training_data_v1.MetaInfo{
			TrainingId: params.ModelID,
			UserId:     metaUserID,
		},
		Pagesize:   pagesize,
		Pos:        pos,
		Since:      sinceQuery,
		SearchType: searchType,
	}

	// The marshal from the grpc record to the rest record should probably be just a byte stream transfer.
	// But for now, I prefer a structural copy, I guess.

	getEMetricsClient, err := trainingData.Client().GetEMetrics(params.HTTPRequest.Context(), query)
	if err != nil {
		trainingData.Close()
		logr.WithError(err).Error("GetEMetrics call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return training_data.NewGetEMetricsUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		return error500(logr, "")
	}

	marr := make([]*restmodels.V1EMetrics, 0, pagesize)

	err = nil
	nRecordsActual := 0
	for ; nRecordsActual < int(pagesize); nRecordsActual++ {
		emetricsRecord, err := getEMetricsClient.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logr.WithError(err).Errorf("Cannot read model definition")
			break
		}

		marr = append(marr, &restmodels.V1EMetrics{
			Meta: &restmodels.V1MetaInfo{
				TrainingID: emetricsRecord.Meta.TrainingId,
				UserID:     emetricsRecord.Meta.UserId,
				Time:       emetricsRecord.Meta.Time,
				Rindex:     emetricsRecord.Meta.Rindex,
			},
			Grouplabel: emetricsRecord.Grouplabel,
			Etimes:     makeRestAnyMapFromGrpcAnyMap(emetricsRecord.GetEtimes()),
			Values:     makeRestAnyMapFromGrpcAnyMap(emetricsRecord.GetValues()),
		})
	}
	trimmedList := marr[0:nRecordsActual]

	response := training_data.NewGetEMetricsOK().WithPayload(&restmodels.V1EMetricsList{
		Models: trimmedList,
	})
	logr.Debug("function exit")
	return response
}

func getLoglines(params training_data.GetLoglinesParams) middleware.Responder {
	logr := logger.LocLogger(logWithLoglinesParams(params))
	logr.Debug("function entry")

	trainingData, err := trainingDataClient.NewTrainingDataClient()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service")
		return error500(logr, "")
	}
	defer trainingData.Close()

	var metaUserID = getUserID(params.HTTPRequest)

	var sinceQuery = ""
	if params.SinceTime != nil {
		sinceQuery = *params.SinceTime
	}
	var pagesize int32 = defaultLogPageSize
	if params.Pagesize != nil {
		pagesize = *params.Pagesize
	}
	var pos int64
	if params.Pos != nil {
		pos = *params.Pos
	}
	var searchType grpc_training_data_v1.Query_SearchType = grpc_training_data_v1.Query_TERM
	if params.SearchType != nil {
		searchType = makeGrpcSearchTypeFromRestSearchType(*params.SearchType)
	}

	query := &grpc_training_data_v1.Query{
		Meta: &grpc_training_data_v1.MetaInfo{
			TrainingId: params.ModelID,
			UserId:     metaUserID,
			Time:       0,
		},
		Pagesize:   pagesize,
		Pos:        pos,
		Since:      sinceQuery,
		SearchType: searchType,
	}

	// The marshal from the grpc record to the rest record should probably be just a byte stream transfer.
	// But for now, I prefer a structural copy, I guess.

	getLogsClient, err := trainingData.Client().GetLogs(params.HTTPRequest.Context(), query)
	if err != nil {
		trainingData.Close()
		logr.WithError(err).Error("GetLogs call failed")
		if grpc.Code(err) == codes.PermissionDenied {
			return training_data.NewGetLoglinesUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
		return error500(logr, "")
	}

	marr := make([]*restmodels.V1LogLine, 0, pagesize)

	err = nil
	nRecordsActual := 0
	for ; nRecordsActual < int(pagesize); nRecordsActual++ {
		logsRecord, err := getLogsClient.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			logr.WithError(err).Errorf("Cannot read model definition")
			break
		}

		marr = append(marr, &restmodels.V1LogLine{
			Meta: &restmodels.V1MetaInfo{
				TrainingID: logsRecord.Meta.TrainingId,
				UserID:     logsRecord.Meta.UserId,
				Time:       logsRecord.Meta.Time,
				Rindex:     logsRecord.Meta.Rindex,
			},
			Line: logsRecord.Line,
		})
	}
	trimmedList := marr[0:nRecordsActual]

	response := training_data.NewGetLoglinesOK().WithPayload(&restmodels.V1LogLinesList{
		Models: trimmedList,
	})
	logr.Debug("function exit")
	return response
}

func patchModel(params models.PatchModelParams) middleware.Responder {
	logr := logger.LocLogger(logWithUpdateStatusParams(params))
	logr.Debugf("patchModel invoked: %v", params.HTTPRequest.Header)

	if params.Payload.Status != "halt" {
		return models.NewPatchModelBadRequest().WithPayload(&restmodels.Error{
			Error:       "Bad request",
			Code:        http.StatusBadRequest,
			Description: "status parameter has incorrect value",
		})
	}

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.Errorf("Cannot create client for trainer service: %s", err.Error())
		error500(logr, "")
	}
	defer trainer.Close()

	_, err = trainer.Client().UpdateTrainingJob(params.HTTPRequest.Context(), &grpc_trainer_v2.UpdateRequest{
		TrainingId: params.ModelID,
		UserId:     getUserID(params.HTTPRequest),
		Status:     grpc_trainer_v2.Status_HALTED,
	})
	//
	if err != nil {
		logr.Errorf("Trainer status update service call failed: %s", err.Error())
		if grpc.Code(err) == codes.NotFound {
			return models.NewPatchModelNotFound().WithPayload(&restmodels.Error{
				Error:       "Not found",
				Code:        http.StatusNotFound,
				Description: "model_id not found",
			})
		}
		if grpc.Code(err) == codes.PermissionDenied {
			return models.NewPatchModelUnauthorized().WithPayload(&restmodels.Error{
				Error:       "Unauthorized",
				Code:        http.StatusUnauthorized,
				Description: "",
			})
		}
	}
	return models.NewPatchModelAccepted().WithPayload(&restmodels.BasicModel{
		ModelID: params.ModelID,
	})
}

func getMetrics(params models.GetMetricsParams) middleware.Responder {
	logsParams := models.GetLogsParams{
		HTTPRequest: params.HTTPRequest,
		ModelID:     params.ModelID,
		Follow:      params.Follow,
		SinceTime:   params.SinceTime,
		Version:     params.Version,
	}
	return getLogsOrMetrics(logsParams, true)
}

//
// Helper functions
//
func error500(log *logger.LocLoggingEntry, description string) middleware.Responder {
	log.Errorf("Returning 500 error: %s", description)
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.WriteHeader(http.StatusInternalServerError)
		payload, _ := json.Marshal(&restmodels.Error{
			Error:       "Internal server error",
			Description: description,
			Code:        http.StatusInternalServerError,
		})
		w.Write(payload)
	})
}

func getUserID(r *http.Request) string {
	return r.Header.Get(mw.UserIDHeader)
}

// Echo the data received on the WebSocket.
func serveLogHandler(trainer trainerClient.TrainerClient, stream grpc_trainer_v2.Trainer_GetTrainedModelLogsClient,
	logr *logger.LocLoggingEntry, cancel context.CancelFunc, isMetrics bool) websocket.Handler {

	return func(ws *websocket.Conn) {
		defer ws.Close()
		defer trainer.Close()
		defer cancel()

		// TODO: The second param should be log.LogCategoryServeLogHandler, but, for the
		// moment, just use the hard coded string, until the code is committed in dlaas-commons.

		logr.Debugf("Going into Recv() loop")
		var onelinebytes []byte
		for {
			var logFrame *grpc_trainer_v2.ByteStreamResponse
			logFrame, err := stream.Recv()
			if err == io.EOF {
				logr.Infof("serveLogHandler stream.Recv() is EOF")
			}
			if logFrame != nil && len(logFrame.Data) > 0 {
				if isMetrics {

					byteReader := bytes.NewReader(logFrame.Data)
					bufioReader := bufio.NewReader(byteReader)
					for {
						lineBytes, err := bufioReader.ReadBytes('\n')
						if lineBytes != nil {
							lenRead := len(lineBytes)

							if err == nil || (err == io.EOF && lenRead > 0 && lineBytes[lenRead-1] == '}') {
								if onelinebytes != nil {
									lineBytes = append(onelinebytes[:], lineBytes[:]...)
									onelinebytes = nil
								}
								// We should just scan for the first non-white space.
								if len(bytes.TrimSpace(lineBytes)) == 0 {
									continue
								}

								ws.Write(lineBytes)
								// Take a short snooze, just to not take over CPU, etc.
								// time.Sleep(time.Millisecond * 250)
							} else {
								if onelinebytes == nil {
									onelinebytes = lineBytes
								} else {
									onelinebytes = append(onelinebytes[:], lineBytes[:]...)
								}
							}
						}
						if err == io.EOF {
							break
						}
					}
					logr.Debug("==== done processing logFrame.Data ====")

				} else {
					var bytes []byte
					bytes = logFrame.Data
					n, errWrite := ws.Write(bytes)
					if errWrite != nil && errWrite != io.EOF {
						logr.WithError(errWrite).Errorf("serveLogHandler Write returned error")
						break
					}
					logr.Debugf("wrote %d bytes", n)
				}
			}

			// either EOF or error reading from trainer
			if err != nil {
				logr.WithError(err).Debugf("Breaking from Recv() loop")
				break
			}
			time.Sleep(time.Millisecond * 2)
		}
	}
}

func getTrainingLogsWS(trainer trainerClient.TrainerClient, params models.GetLogsParams,
	stream grpc_trainer_v2.Trainer_GetTrainedModelLogsClient,
	cancel context.CancelFunc, isMetrics bool) middleware.Responder {

	logr := logger.LocLogger(logWithGetLogsParams(params))
	logr.Debugf("Setting up web socket: %v", params.HTTPRequest.Header)

	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		logr.Debugf("In responderFunc")
		serveLogHandler(trainer, stream, logr, cancel, isMetrics).ServeHTTP(w, params.HTTPRequest)
	})
}

func serveLogHandlerTest(logr *logger.LocLoggingEntry, cancel context.CancelFunc) websocket.Handler {
	logr.Debugf("In serveLogHandlerTest")

	return func(ws *websocket.Conn) {
		defer ws.Close()
		defer cancel()
		for i := 0; i < 30; i++ {
			currentTime := time.Now().Local()
			currentTimeString := currentTime.Format("  Sat Mar 7 11:06:39 EST 2015\n")
			logr.Debugf("In websocket test function, writing %s", currentTimeString)
			_, writeStringError := io.WriteString(ws, currentTimeString)
			if writeStringError != nil {
				logr.WithError(writeStringError).Debug("WriteString failed")
				break
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func getTrainingLogsWSTest(ctx context.Context, params models.GetLogsParams,
	cancel context.CancelFunc) middleware.Responder {

	logr := logger.LocLogger(logWithGetLogsParams(params))
	logr.Debugf("Setting up web socket test: %v", params.HTTPRequest.Header)

	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		serveLogHandlerTest(logr, cancel).ServeHTTP(w, params.HTTPRequest)
	})

}

func createModel(req *http.Request, job *grpc_trainer_v2.Job) *restmodels.Model {
	memUnit := job.Training.Resources.MemoryUnit.String()

	var metrics []*restmodels.MetricData

	if job.Metrics != nil {
		metrics = make([]*restmodels.MetricData, 1)

		// The mismatch between the rest-api metrics struct and the gRPC metrics struct
		// causes pain...
		interfaceValues := make(map[string]interface{})

		for k, v := range job.Metrics.Values {
			interfaceValues[k] = v
		}

		newMetrics := &restmodels.MetricData{
			Timestamp: job.Metrics.Timestamp,
			Type:      job.Metrics.Type,
			Iteration: job.Metrics.Iteration,
			Values:    interfaceValues,
		}
		metrics[0] = newMetrics
	} else {
		metrics = nil
	}

	m := &restmodels.Model{
		BasicNewModel: restmodels.BasicNewModel{
			BasicModel: restmodels.BasicModel{
				ModelID: job.TrainingId,
			},
			Location: req.URL.Path + "/" + job.TrainingId,
		},
		Name:        job.ModelDefinition.Name,
		Description: job.ModelDefinition.Description,
		Framework: &restmodels.Framework{
			Name:    job.ModelDefinition.Framework.Name,
			Version: job.ModelDefinition.Framework.Version,
		},
		Metrics: metrics,
		Training: &restmodels.Training{
			Command:    job.Training.Command,
			Cpus:       float64(job.Training.Resources.Cpus),
			Gpus:       float64(job.Training.Resources.Gpus),
			Memory:     float64(job.Training.Resources.Memory),
			MemoryUnit: &memUnit,
			Learners:   job.Training.Resources.Learners,
			InputData:  job.Training.InputData,
			OutputData: job.Training.OutputData,
			TrainingStatus: &restmodels.TrainingStatus{
				Status:            job.Status.Status.String(),
				StatusDescription: job.Status.Status.String(),
				StatusMessage:     job.Status.StatusMessage,
				ErrorCode:         job.Status.ErrorCode,
				Submitted:         job.Status.SubmissionTimestamp,
				Completed:         job.Status.CompletionTimestamp,
			},
		},
	}

	// add data stores
	for i, v := range job.Datastores {
		m.DataStores = append(m.DataStores, &restmodels.Datastore{
			DataStoreID: v.Id,
			Type:        v.Type,
			Connection:  v.Connection,
		})
		for k, v := range job.Datastores[i].Fields {
			m.DataStores[i].Connection[k] = v
		}
	}

	return m
}
