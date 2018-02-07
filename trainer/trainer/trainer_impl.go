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

package trainer

import (
	"bufio"
	"fmt"
	"io"

	"google.golang.org/grpc/status"
	"gopkg.in/mgo.v2"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	"github.com/nu7hatch/gouuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ventu-io/go-shortid"
	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/metricsmon"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	"github.ibm.com/ffdl/ffdl-core/commons/service/client"

	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
	trainerClient "github.ibm.com/ffdl/ffdl-core/trainer/client"
	tdsClient "github.ibm.com/ffdl/ffdl-core/metrics/client"
	tdsService "github.ibm.com/ffdl/ffdl-core/metrics/service/grpc_training_data_v1"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"

	"sync"
	"time"

	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	"errors"

	"github.ibm.com/ffdl/ffdl-core/trainer/storage"
)

const internalObjectStoreID = "dlaas_internal_os"

const (
	modelsBucketKey        = "objectstore.bucket.models"
	trainedModelsBucketKey = "objectstore.bucket.trainedmodels"

	defaultModelsBucket        = "dlaas-models"
	defaultTrainedModelsBucket = "dlaas-trained-models"

	debugLogsMode = false
)

// Confuse `go vet' to not check this `Errorf' call. :(
// See https://github.com/grpc/grpc-go/issues/90
var gerrf = status.Errorf

// Service represents the functionality of the trainer service
type Service interface {
	grpc_trainer_v2.TrainerServer
	service.LifecycleHandler
}

type trainerMetrics struct {
	createTrainingJobCounter, deleteTrainingJobCounter,
	downloadTrainedModelJobCounter, downloadTrainingMetricsJobCounter,
	rateLimitTrainingJobCounter,
	trainingJobFailedCounter, trainingJobSucceededCounter metrics.Counter
}

type trainerService struct {
	mtx                 sync.RWMutex
	datastore           storage.DataStore
	lcm                 client.LcmClient
	repo                repository
	modelsBucket        string
	trainedModelsBucket string
	metrics             *trainerMetrics
	tds                 tdsClient.TrainingDataClient
	service.Lifecycle
}

// NewService creates a new trainer service.
func NewService() Service {
	logr := logger.LogServiceBasic(logger.LogkeyTrainerService)

	config.FatalOnAbsentKey("mongo.address")
	trainerMetrics := trainerMetrics{
		createTrainingJobCounter:          metricsmon.NewCounter("trainer_trainings_create_total", "Metrics for total rate limit invocations on trainer", []string{"framework", "version", "gpus", "cpus", "memory"}),
		deleteTrainingJobCounter:          metricsmon.NewCounter("trainer_trainings_delete_total", "Metrics for total rate limit invocations on trainer", []string{}),
		downloadTrainedModelJobCounter:    metricsmon.NewCounter("trainer_model_download_total", "Metrics for total rate limit invocations on trainer", []string{}),
		downloadTrainingMetricsJobCounter: metricsmon.NewCounter("trainer_metrics_download_total", "Metrics for total rate limit invocations on trainer", []string{}),
		rateLimitTrainingJobCounter:       metricsmon.NewCounter("trainer_ratelimitinvocations_total", "Metrics for total rate limit invocations on trainer", []string{}),
		trainingJobFailedCounter:          metricsmon.NewCounter("trainer_trainings_failed_total", "Metrics for failed training jobs", []string{"framework", "version", "gpus", "cpus", "memory", "type", "errorcode"}),
		trainingJobSucceededCounter:       metricsmon.NewCounter("trainer_trainings_success_total", "Metrics for succeeded training jobs", []string{"framework", "version", "gpus", "cpus", "memory"}),
	}

	ds, err := storage.CreateDataStore(config.GetDataStoreType(), config.GetDataStoreConfig())
	if err != nil {
		logr.WithError(err).Fatalf("Cannot create datastore")
	}
	err = ds.Connect()
	if err != nil {
		logr.WithError(err).Fatalf("Cannot connect to object store")
	}

	repo, err := newTrainingsRepository(viper.GetString("mongo.address"),
		viper.GetString("mongo.database"), viper.GetString("mongo.username"),
		viper.GetString("mongo.password"), config.GetMongoCertLocation(), "training_jobs")
	if err != nil {
		logr.WithError(err).Fatalf("Cannot create repository with %s %s %s", viper.GetString("mongo.address"), viper.GetString("mongo.database"), viper.GetString("mongo.username"))
	}

	s := &trainerService{
		datastore:           ds,
		repo:                repo,
		modelsBucket:        getModelsBucket(),
		trainedModelsBucket: getTrainedModelsBucket(),
		metrics:             &trainerMetrics,
	}
	logr.Infof("Bucket for model definitions: %s", s.modelsBucket)
	logr.Infof("Bucket for trained models: %s", s.trainedModelsBucket)

	s.RegisterService = func() {
		grpc_trainer_v2.RegisterTrainerServer(s.Server, s)
	}
	return s
}

// NewTestService creates a new service instance for testing
func NewTestService(ds storage.DataStore, repo repository,
	lcm client.LcmClient, tds tdsClient.TrainingDataClient) Service {

	trainerMetrics := trainerMetrics{
		createTrainingJobCounter:          discard.NewCounter(),
		deleteTrainingJobCounter:          discard.NewCounter(),
		downloadTrainedModelJobCounter:    discard.NewCounter(),
		downloadTrainingMetricsJobCounter: discard.NewCounter(),
		rateLimitTrainingJobCounter:       discard.NewCounter(),
		trainingJobFailedCounter:          discard.NewCounter(),
		trainingJobSucceededCounter:       discard.NewCounter(),
	}

	s := &trainerService{
		datastore:           ds,
		repo:                repo,
		lcm:                 lcm,
		modelsBucket:        getModelsBucket(),
		trainedModelsBucket: getTrainedModelsBucket(),
		metrics:             &trainerMetrics,
		tds:                 tds,
	}

	s.RegisterService = func() {
		grpc_trainer_v2.RegisterTrainerServer(s.Server, s)
	}
	return s
}

func debugLogger(logrr *logrus.Entry, isEnabled bool) *logger.LocLoggingEntry {
	logr := new(logger.LocLoggingEntry)
	logr.Logger = logrr
	logr.Enabled = isEnabled

	return logr
}

// Cover for deprecated grpc function.
func grpcErrorDesc(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}

func (s *trainerService) CreateTrainingJob(ctx context.Context, req *grpc_trainer_v2.CreateRequest) (*grpc_trainer_v2.CreateResponse, error) {
	sid, _ := shortid.Generate()
	id := fmt.Sprintf("training-%s", sid)
	logr := logger.LocLogger(logWith(id, req.UserId))

	if err := s.validateRequest(logr.Logger, req); err != nil {
		return nil, err
	}

	limit := config.GetResourceLimit()
	gpusRequested := req.Training.Resources.Gpus
	if gpusRequested > 0 && limit > 0 { //if limit is missing or 0 then don't rate limit
		logr.Debugf("Executing resource limit check since the resource limit is set to %d", limit)
		if records, err := s.repo.FindCurrentlyRunningTrainings(config.GetResourceLimitQuerySize()); err == nil && len(records) > 0 {
			if rateLimitTrainingJob(records, limit, logr) {
				s.metrics.rateLimitTrainingJobCounter.Add(1)
				err := gerrf(codes.ResourceExhausted, "No more additional trainings can be scheduled at this time. Please try later")
				logr.WithError(err).Warnf("Rejecting request for create training job, because exceeded resource limit")
				return nil, err
			}
		} else {
			logr.WithError(err).Warnf("did not execute rate limiting correctly, returned number of records count is %d", len(records))
		}
	}

	//request is validated, now bump up the counter
	s.metrics.createTrainingJobCounter.With("framework", req.ModelDefinition.Framework.Name,
		"version", req.ModelDefinition.Framework.Version,
		"gpus", strconv.Itoa(int(req.Training.Resources.Gpus)),
		"cpus", strconv.Itoa(int(req.Training.Resources.Cpus)),
		"memory", strconv.Itoa(int(req.Training.Resources.Memory))).Add(1)

	// we assume only one training input and output data at this point
	inputDatastore := findDatastore(req.Training.InputData[0], req.Datastores)
	outputDatastore := s.getOutputDatastore(req.Training.OutputData, req.Datastores)

	// upload model definition ZIP file to object store and set location
	if req.ModelDefinition.Content != nil {
		err := s.datastore.UploadArchive(s.modelsBucket, getModelZipFileName(id), req.ModelDefinition.Content)
		if err != nil {
			logr.Errorf("Error uploading model to object store: %s", err.Error())
			return nil, err
		}
		req.ModelDefinition.Location = fmt.Sprintf("%s/%s.zip", s.modelsBucket, id)
	}

	setDefaultResourceRequirements(req.Training)

	jobConfig, err := createJobConfig(id, req, inputDatastore, outputDatastore)
	if err != nil {
		logr.Errorf("Failed to create job config: %s", err.Error())
		return nil, err
	}

	// create a copy of the model definition without the content field (do not store it to the database)
	modelWithoutContent := *req.ModelDefinition
	modelWithoutContent.Content = nil

	tr := &TrainingRecord{
		TrainingID:      id,
		UserID:          req.UserId,
		JobID:           jobConfig.Name, // TODO this need to be removed!
		ModelDefinition: &modelWithoutContent,
		Training:        req.Training,
		Datastores:      req.Datastores,
		TrainingStatus: &grpc_trainer_v2.TrainingStatus{
			Status:              grpc_trainer_v2.Status_PENDING,
			SubmissionTimestamp: fmt.Sprintf("%s", time.Now()),
		},
		Metrics: nil,
	}

	err = s.repo.Store(tr)
	if err != nil {
		logr.Errorf("Failed to resolve output datastore: %s", err.Error())
		return nil, gerrf(codes.Internal, grpcErrorDesc(err))
	}

	lcm, err := s.lcmClient()
	if err != nil {
		logr.Errorf("Cannot create LCM service client: %s", err.Error())
		return nil, err
	}
	defer lcm.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = lcm.Client().DeployTrainingJob(ctx, jobConfig)
	if err != nil {
		logr.Errorf("Cannot deploy training job with id %s: %s", jobConfig.Name, err.Error())
		return nil, err
	}

	return &grpc_trainer_v2.CreateResponse{TrainingId: id}, nil
}

func (s *trainerService) GetTrainingJob(ctx context.Context, req *grpc_trainer_v2.GetRequest) (*grpc_trainer_v2.GetResponse, error) {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))

	tr, err := s.repo.Find(req.TrainingId)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, gerrf(codes.NotFound, "Training with id %s not found.", req.TrainingId)
		}
		logr.WithError(err).Errorf("Cannot retrieve training record")
		return nil, err
	}

	if tr.UserID != req.UserId {
		msg := fmt.Sprint("User does not have permission to read training data")
		logr.Error(msg)
		return nil, gerrf(codes.PermissionDenied, msg)
	}
	jobb := &grpc_trainer_v2.Job{
		UserId:          tr.UserID,
		JobId:           tr.JobID,
		ModelDefinition: tr.ModelDefinition,
		TrainingId:      tr.TrainingID,
		Training:        tr.Training,
		Status:          tr.TrainingStatus,
		Datastores:      tr.Datastores,
		Metrics:         tr.Metrics,
	}
	return &grpc_trainer_v2.GetResponse{
		Job: jobb,
	}, nil
}

func (s *trainerService) GetTrainingStatusID(ctx context.Context, req *grpc_trainer_v2.GetRequest) (*grpc_trainer_v2.GetStatusIDResponse, error) {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))

	statusID, err := s.repo.FindTrainingStatusID(req.TrainingId)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, gerrf(codes.NotFound, "Training with id %s not found.", req.TrainingId)
		}
		logr.WithError(err).Errorf("Cannot retrieve record for training %s", req.TrainingId)
		return nil, err
	}
	return &grpc_trainer_v2.GetStatusIDResponse{
		Status: statusID,
	}, nil
}

func (s *trainerService) UpdateTrainingJob(ctx context.Context, req *grpc_trainer_v2.UpdateRequest) (*grpc_trainer_v2.UpdateResponse, error) {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Debugf("UpdateTrainingJob called for training %s", req.TrainingId)
	s.mtx.Lock()
	defer s.mtx.Unlock()

	training, err := s.repo.Find(req.TrainingId)
	if err != nil {
		logr.Errorf("Cannot retrieve training ''%s': %s", req.TrainingId, err.Error())
		return nil, err
	}
	if training == nil {
		// training does not exist
		return nil, gerrf(codes.NotFound, "Training with id %s not found.", req.TrainingId)
	}

	if training.UserID != req.UserId {
		msg := fmt.Sprintf("User %s does not have permission to update training data with id %s.", req.UserId, req.TrainingId)
		logr.Error(msg)
		return nil, gerrf(codes.PermissionDenied, msg)
	}

	ts := training.TrainingStatus
	ts.Status = req.Status
	ts.StatusMessage = req.StatusMessage
	ts.ErrorCode = req.ErrorCode

	if req.Status == grpc_trainer_v2.Status_COMPLETED || req.Status == grpc_trainer_v2.Status_FAILED || req.Status == grpc_trainer_v2.Status_HALTED {
		ts.CompletionTimestamp = fmt.Sprintf("%s", time.Now())
		if req.Timestamp > 0 {
			ts.CompletionTimestamp = fmt.Sprintf("%s", req.Timestamp)
		}
	}
	if req.Status == grpc_trainer_v2.Status_DOWNLOADING {
		ts.DownloadStartTimestamp = fmt.Sprintf("%s", time.Now())
		if req.Timestamp > 0 {
			ts.DownloadStartTimestamp = fmt.Sprintf("%s", req.Timestamp)
		}
	}
	if req.Status == grpc_trainer_v2.Status_PROCESSING {
		ts.ProcessStartTimestamp = fmt.Sprintf("%s", time.Now())
		if req.Timestamp > 0 {
			ts.ProcessStartTimestamp = fmt.Sprintf("%s", req.Timestamp)
		}
	}
	if req.Status == grpc_trainer_v2.Status_STORING {
		ts.StoreStartTimestamp = fmt.Sprintf("%s", time.Now())
		if req.Timestamp > 0 {
			ts.StoreStartTimestamp = fmt.Sprintf("%s", req.Timestamp)
		}
	}

	// send monitoring metrics for failed/succeeded jobs
	if req.Status == grpc_trainer_v2.Status_COMPLETED || req.Status == grpc_trainer_v2.Status_FAILED {
		counter := s.metrics.trainingJobSucceededCounter
		if req.Status == grpc_trainer_v2.Status_FAILED {
			errorType := "server"
			if strings.HasPrefix(req.ErrorCode, "C") {
				errorType = "client"
			}
			counter = s.metrics.trainingJobFailedCounter.With("type", errorType, "errorcode", req.ErrorCode)
		}
		counter.With("framework", training.ModelDefinition.Framework.Name,
			"version", training.ModelDefinition.Framework.Version,
			"gpus", strconv.Itoa(int(training.Training.Resources.Gpus)),
			"cpus", strconv.Itoa(int(training.Training.Resources.Cpus)),
			"memory", strconv.Itoa(int(training.Training.Resources.Memory))).Add(1)
	}

	err = s.repo.Store(training)
	if err != nil {
		logr.Errorf("Failed updating status of training %s in DB: %s", req.TrainingId, err.Error())
		return nil, err
	}

	training, err = s.repo.Find(req.TrainingId)
	if err != nil {
		logr.Errorf("Cannot retrieve training '%s': %s", req.TrainingId, err.Error())
		return nil, err
	}
	if training == nil {
		// training does not exist
		return nil, gerrf(codes.NotFound, "Training with id %s not found.", req.TrainingId)
	}
	ts = training.TrainingStatus
	logr.Debugf("CHECKING Stored training %s, Status %s Error Code %s Message %s", req.TrainingId, ts.Status, ts.ErrorCode, ts.StatusMessage)

	return &grpc_trainer_v2.UpdateResponse{TrainingId: training.TrainingID}, nil
}

func (s *trainerService) GetAllTrainingsJobs(ctx context.Context, req *grpc_trainer_v2.GetAllRequest) (*grpc_trainer_v2.GetAllResponse, error) {
	logr := logger.LocLogger(logEntry().WithField(logger.LogkeyUserID, req.UserId))
	logr.Debugf("GetAllTrainingsJobs called")

	jobs, err := s.repo.FindAll(req.UserId)
	if err != nil {
		msg := "Failed to retrieve all training jobs"
		logr.WithError(err).Errorf(msg)
		return nil, gerrf(codes.Internal, msg)
	}
	resp := &grpc_trainer_v2.GetAllResponse{
		Jobs: make([]*grpc_trainer_v2.Job, len(jobs)),
	}
	for i, job := range jobs {
		resp.Jobs[i] = &grpc_trainer_v2.Job{
			UserId:          job.UserID,
			JobId:           job.JobID,
			ModelDefinition: job.ModelDefinition,
			TrainingId:      job.TrainingID,
			Training:        job.Training,
			Status:          job.TrainingStatus,
			Datastores:      job.Datastores,
		}
	}
	return resp, nil
}

// cover for depreciated grpc method
func grpcCode(err error) codes.Code {
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}
	return codes.Unknown
}

func (s *trainerService) deleteJobFromTDS(query *tdsService.Query, logr *logger.LocLoggingEntry) error {
	tds, err := s.tdsClient()
	if err != nil {
		logr.WithError(err).Error("Cannot create TDS client")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	delResponse, err := tds.Client().DeleteJob(ctx, query)
	if err != nil {
		logr.WithError(err).Error("tds DeleteJob returned error")
		return err
	}
	if !delResponse.Success {
		logr.Warn("tds DeleteJob reported false for success")
	}
	return nil
}

func (s *trainerService) DeleteTrainingJob(ctx context.Context,
	req *grpc_trainer_v2.DeleteRequest) (*grpc_trainer_v2.DeleteResponse, error) {

	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Debugf("DeleteTrainingJob called")

	s.metrics.deleteTrainingJobCounter.Add(1)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	readResp, err := s.GetTrainingJob(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})

	if err != nil {
		logr.WithError(err).Errorf("Failing querying training job")
		return nil, err
	}

	err = s.deleteJobFromTDS(&tdsService.Query{
		Meta: &tdsService.MetaInfo{
			TrainingId: req.TrainingId,
			UserId:     req.UserId,
		},
	}, logr)
	if err != nil {
		logr.WithError(err).Warn("deleteJobFromTDS returned error")
	}

	var job *grpc_trainer_v2.Job
	if readResp != nil {
		job = readResp.Job

		// delete the job if exists
		lcm, err := s.lcmClient()
		if err != nil {
			logr.WithError(err).Errorln("Cannot create lcm service client")
			return nil, err
		}
		defer lcm.Close()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err = lcm.Client().KillTrainingJob(ctx, &service.JobKillRequest{
			Name:       job.JobId,
			TrainingId: job.TrainingId,
			UserId:     job.UserId,
		})

		// tolerate "not found" because it just means the job is no longer running
		if err != nil && grpcCode(err) != codes.NotFound {
			logr.WithError(err).Errorf("Failed to kill job '%s'", job.JobId)
			return nil, err
		}
		logr.Debugf("Kubernetes job '%s' does not longer exist.", job.JobId)

		// delete model content from data store
		err = s.datastore.DeleteArchive(s.modelsBucket, getModelZipFileName(job.JobId))
		if err != nil {
			logr.Errorf("Error deleting model from object store: %s", err.Error())
			// log this error, but continue with deleting the training record anyway
		}

		// delete from DB
		err = s.repo.Delete(job.TrainingId)
		if err != nil {
			logr.WithError(err).Errorf("Failed to delete training job '%s' from database", job.TrainingId)
			return nil, err
		}
		return &grpc_trainer_v2.DeleteResponse{TrainingId: job.JobId}, nil
	}
	return nil, gerrf(codes.NotFound, "Training with id '%s' not found.", req.TrainingId)
}

func (s *trainerService) HaltTrainingJob(ctx context.Context, req *grpc_trainer_v2.HaltRequest) (*grpc_trainer_v2.HaltResponse, error) {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Debugf("HaltTrainingJob called")
	return nil, gerrf(codes.Unimplemented, "ResumeTrainingJob not implemented yet")
}

func (s *trainerService) ResumeTrainingJob(ctx context.Context, req *grpc_trainer_v2.ResumeRequest) (*grpc_trainer_v2.ResumeResponse, error) {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Debugf("HaltTrainingJob called")
	return nil, gerrf(codes.Unimplemented, "ResumeTrainingJob not implemented yet")
}

func (s *trainerService) GetModelDefinition(req *grpc_trainer_v2.ModelDefinitionRequest, stream grpc_trainer_v2.Trainer_GetModelDefinitionServer) error {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Infof("GetTrainedModel")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := s.GetTrainingJob(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})
	if err != nil {
		logr.WithError(err).Errorf("Failed to read training with id: %s", req.TrainingId)
		return gerrf(codes.Internal, "Failed to read training with id: %s", req.TrainingId)
	}
	if resp == nil || resp.Job == nil {
		return gerrf(codes.NotFound, "Training with id '%s' not found.", req.TrainingId)
	}

	// TODO we need to change this to accept a writer to be more efficient
	payload, err := s.datastore.DownloadArchive(s.modelsBucket, getModelZipFileName(req.TrainingId))
	if err != nil {
		logr.Errorf("Downloading model definition archive failed: %s", err)
	}
	err = stream.Send(&grpc_trainer_v2.ZippedDataChunk{
		Data: payload,
	})
	if err != nil {
		logr.WithError(err).Errorf("Failed to send zipped chunk.")
		return err
	}
	return nil
}

func (s *trainerService) GetTrainedModel(req *grpc_trainer_v2.TrainedModelRequest, stream grpc_trainer_v2.Trainer_GetTrainedModelServer) error {
	//s.mtx.Lock()
	//defer s.mtx.Unlock()

	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	logr.Infof("GetTrainedModel")

	s.metrics.downloadTrainedModelJobCounter.Add(1)
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := s.GetTrainingJob(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})
	if err != nil {
		logr.WithError(err).Errorf("Error reading training with id: %s", req.TrainingId)
		return err
	}
	if resp == nil || resp.Job == nil {
		return gerrf(codes.NotFound, "Training with id '%s' not found.", req.TrainingId)
	}

	var ostore storage.DataStore
	ds := s.getOutputDatastore(resp.Job.Training.OutputData, resp.Job.Datastores)
	ostore, err = storage.CreateDataStore(ds.Type, ds.Connection)
	if err != nil {
		logr.WithError(err).Errorf("Error creating datastore: %v", ds)
		return err
	}
	if err := ostore.Connect(); err != nil {
		logr.WithError(err).Error("Error connect to datastore")
		return err
	}
	defer ostore.Disconnect()

	trainedModelSize, err := ostore.GetTrainedModelSize(fmt.Sprintf("%s/%s", ds.Fields["bucket"], resp.Job.TrainingId),
		resp.Job.Training.Resources.Learners)

	if err != nil {
		logr.WithError(err).Error("Error retrieving trained model size")
		return err
	}
	logr.Debugf("The size of the trained model is %d", trainedModelSize)

	// DP only allows downloads of sizes less than 200MBs
	if trainedModelSize > 200000000 {
		logr.Debugf("Trained model for '%s' exceeded download limit size.", req.TrainingId)
		return gerrf(codes.FailedPrecondition,
			"Trained model exceeded download limit. Download from your cloud storage directly")
	}

	r, w := io.Pipe() // connect I/O without temp space.

	go func() {
		// write to pipe by downloading
		err := ostore.DownloadTrainedModelAsZipStream(fmt.Sprintf("%s/%s", ds.Fields["bucket"], resp.Job.TrainingId),
			resp.Job.Training.Resources.Learners, w)

		if err != nil {
			logr.WithError(err).Error("Downloading trained model failed")
			w.CloseWithError(err)
		}
		if err := w.Close(); err != nil {
			logr.WithError(err).Error("Closing writer failed")
		}
	}()

	reader := bufio.NewReader(r)
	buf := make([]byte, 0, 10*1024)
	for {
		n, err := reader.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				//logr.Errorf("Downloading trained model failed: %s", err.Error())
				break
			}
			return err
		}
		// process buf
		if err != nil && err != io.EOF {
			logr.Errorf("Downloading trained model failed: %s", err.Error())
			return err
		}
		err = stream.Send(&grpc_trainer_v2.ZippedDataChunk{
			Data: buf,
		})
		if err != nil {
			logr.WithError(err).Error("Failed to send zipped data chunk")
			return err
		}
	}
	return nil
}

func (s *trainerService) GetTrainedModelLogsFromObjStore(req *grpc_trainer_v2.TrainedModelLogRequest,
	stream grpc_trainer_v2.Trainer_GetTrainedModelLogsFromObjStoreServer) error {

	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := s.GetTrainingJob(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})
	if err != nil {
		logr.WithError(err).Debugf("Error reading training with id: %s", req.TrainingId)
		return err
	}
	if resp == nil || resp.Job == nil {
		return gerrf(codes.NotFound, "Training with id '%s' not found.", req.TrainingId)
	}

	var ostore storage.DataStore
	logr.Debugf("Training output data: %v", resp.Job.Training.OutputData)
	logr.Debugf("Training datastores: %v", resp.Job.Datastores)

	ds := s.getOutputDatastore(resp.Job.Training.OutputData, resp.Job.Datastores) // TODO add some checks
	ostore, err = storage.CreateDataStore(ds.Type, ds.Connection)
	if err != nil {
		logr.WithError(err).Errorf("Error creating data store '%s', properties: %v", ds.Type, ds.Connection)
		return err
	}
	if err := ostore.Connect(); err != nil {
		logr.WithError(err).Errorf("Error connecting to data store '%s', properties: %v", ds.Type, ds.Connection)
		return err
	}
	defer ostore.Disconnect()

	var logFileName string
	if req.IsMetrics {
		if req.IsSummary {
			logFileName = "summary-metrics.txt"
		} else {
			logFileName = "evaluation-metrics.txt"
		}
	} else {
		logFileName = "training-log.txt"
	}

	r, w := io.Pipe() // connect I/O without temp space.
	go func() {
		// write to pipe by downloading
		err := ostore.DownloadTrainedModelLogFile(fmt.Sprintf("%s/%s", ds.Fields["bucket"], resp.Job.TrainingId),
			0, 1, logFileName, w)
		w.CloseWithError(err)
	}()

	reader := bufio.NewReader(r)
	buf := make([]byte, 0, 5*1024)
	for {
		n, err := reader.Read(buf[:cap(buf)])
		buf = buf[:n]
		if n == 0 {
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			logr.WithError(err).Debugf("reader.Read(...) returned error")
			return err
		}
		// process buf
		if err != nil && err != io.EOF {
			logr.WithError(err).Debugf("reader.Read(...) returned error")
			return err
		}
		err = stream.Send(&grpc_trainer_v2.ByteStreamResponse{
			Data: buf,
		})
		if err != nil {
			logr.WithError(err).Debugf("stream.Send(...) returned error")
			return err
		}
	}
	return nil
}

func (s *trainerService) GetTrainedModelLogs(req *grpc_trainer_v2.TrainedModelLogRequest, outStream grpc_trainer_v2.Trainer_GetTrainedModelLogsServer) error {
	logr := logger.LocLogger(logWith(req.TrainingId, req.UserId))

	lcm, err := s.lcmClient()
	if err != nil {
		logr.WithError(err).Error("Cannot create LCM service client")
		return err
	}
	defer lcm.Close()

	var ctx context.Context
	var cancel context.CancelFunc
	logr.Debugf("follow is %t", req.Follow)
	if req.Follow {
		ctx, cancel = context.WithTimeout(context.Background(), 10*(time.Hour*24))
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	infosRequest := &service.TrainerContainerInfosRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
		Follow:     req.Follow,
		Metrics:    false,
		Summary:    false,
	}
	inStream, err := lcm.Client().GetTrainingLogStream(ctx, infosRequest)

	if err != nil {
		logr.WithError(err).Debugf("Error reading training log")
		return err
	}

	if inStream == nil {
		logr.WithError(err).Debugf("GetTrainingLogStream strangely returned nil")
		return fmt.Errorf("GetTrainingLogStream strangely returned nil")
	}

	for {
		chunk, err := inStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logr.WithError(err).Errorf("cannot read trained model log")
			return fmt.Errorf("cannot read trained model log: %v", err)
		}
		errSend := outStream.Send(&grpc_trainer_v2.ByteStreamResponse{Data: chunk.Data})
		if errSend != nil {
			logr.WithError(errSend).Errorf("cannot send trained model log")
			return fmt.Errorf("cannot send trained model log: %v", err)
		}
	}

	return nil
}

func (s *trainerService) GetTrainedModelMetrics(req *grpc_trainer_v2.TrainedModelMetricsRequest,
	outStream grpc_trainer_v2.Trainer_GetTrainedModelMetricsServer) error {

	return errors.New(
		"GetTrainedModelMetrics no longer supported, use GetTrainingEMetrics instead")
}

func marshalQuerySearchType(st grpc_trainer_v2.Query_SearchType) tdsService.Query_SearchType {
	searchType := tdsService.Query_TERM

	switch st {
	case grpc_trainer_v2.Query_TERM:
		searchType = tdsService.Query_TERM
		break
	case grpc_trainer_v2.Query_NESTED:
		searchType = tdsService.Query_NESTED
		break
	case grpc_trainer_v2.Query_MATCH:
		searchType = tdsService.Query_MATCH
		break
	case grpc_trainer_v2.Query_ALL:
		searchType = tdsService.Query_ALL
		break
	}
	return searchType
}

func marshalTDSQueryToTrainerQuery(in *grpc_trainer_v2.Query) *tdsService.Query {
	query := &tdsService.Query{
		Meta: &tdsService.MetaInfo{
			TrainingId: in.Meta.TrainingId,
			UserId: in.Meta.UserId,
			Time: in.Meta.Time,
			Rindex: in.Meta.Rindex,
			//Subid: in.Meta.Subid,
		},
		Pos:        in.Pos,
		Pagesize:   in.Pagesize,
		Since:      in.Since,
		SearchType: marshalQuerySearchType(in.SearchType),
	}
	return query
}

func (s *trainerService) GetTrainingLogs(in *grpc_trainer_v2.Query,
	outStream grpc_trainer_v2.Trainer_GetTrainingLogsServer) error {

	logr := logger.LocLogger(logWith(in.Meta.TrainingId, in.Meta.UserId))

	//noinspection GoBoolExpressions
	dlogr := debugLogger(logr.Logger, debugLogsMode)

	dlogr.Debug("entry")

	tds, err := s.tdsClient()
	if err != nil {
		logr.WithError(err).Error("Cannot create LCM service client")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	dlogr.Debugf("Query to send from client: %+v", in)

	query := marshalTDSQueryToTrainerQuery(in)

	dlogr.Debugf("Query to send to training-data: %+v", query)

	inStream, err := tds.Client().GetLogs(ctx, query)

	for {
		dlogr.Debugf("inStream.Recv()")
		chunk, err := inStream.Recv()
		if err == io.EOF {
			dlogr.Debug("eof")
			break
		}
		if err != nil {
			logr.WithError(err).Errorf("cannot read trained model log")
			return fmt.Errorf("cannot read trained model log: %v", err)
		}
		dlogr.Debugf("sending line: %d", chunk.Meta.Rindex)
		errSend := outStream.Send(&grpc_trainer_v2.LogLine{
			Meta: &grpc_trainer_v2.MetaInfo{
				TrainingId: chunk.Meta.TrainingId,
				UserId: chunk.Meta.UserId,
				Time: chunk.Meta.Time,
				Rindex: chunk.Meta.Rindex,
				//Subid: chunk.Meta.Subid,
			},
			Line: chunk.Line,
		})
		if errSend != nil {
			logr.WithError(errSend).Errorf("cannot send trained model log")
			return fmt.Errorf("cannot send trained model log: %v", err)
		}
		dlogr.Debugf("sent without error")
	}
	dlogr.Debug("exit with nil return")
	return nil
}

func marshalTDSDataType2TrainerDataType(dt tdsService.Any_DataType) grpc_trainer_v2.Any_DataType {
	dataType := grpc_trainer_v2.Any_STRING

	switch dt {
	case tdsService.Any_STRING:
		dataType = grpc_trainer_v2.Any_STRING
		break
	case tdsService.Any_JSONSTRING:
		dataType = grpc_trainer_v2.Any_JSONSTRING
		break
	case tdsService.Any_INT:
		dataType = grpc_trainer_v2.Any_INT
		break
	case tdsService.Any_FLOAT:
		dataType = grpc_trainer_v2.Any_FLOAT
		break
	}
	return dataType
}

func marshalTDSMapToTrainerMap(tdsMap map[string]*tdsService.Any) map[string]*grpc_trainer_v2.Any {
	grpcMap := make(map[string]*grpc_trainer_v2.Any)
	for k, v := range tdsMap {
		trainerDT := marshalTDSDataType2TrainerDataType(v.Type)
		grpcMap[k] = &grpc_trainer_v2.Any{Type: trainerDT, Value: v.Value}
	}
	return grpcMap
}

func (s *trainerService) GetTrainingEMetrics(in *grpc_trainer_v2.Query,
	outStream grpc_trainer_v2.Trainer_GetTrainingEMetricsServer) error {

	logr := logger.LocLogger(logWith(in.Meta.TrainingId, in.Meta.UserId))
	tds, err := s.tdsClient()
	if err != nil {
		logr.WithError(err).Error("Cannot create LCM service client")
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*4)
	defer cancel()

	query := marshalTDSQueryToTrainerQuery(in)

	inStream, err := tds.Client().GetEMetrics(ctx, query)

	for {
		chunk, err := inStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logr.WithError(err).Errorf("cannot read trained model log")
			return fmt.Errorf("cannot read trained model log: %v", err)
		}
		errSend := outStream.Send(&grpc_trainer_v2.EMetrics{
			Meta: &grpc_trainer_v2.MetaInfo{
				TrainingId: chunk.Meta.TrainingId,
				UserId: chunk.Meta.UserId,
				Time: chunk.Meta.Time,
				Rindex: chunk.Meta.Rindex,
				//Subid: chunk.Meta.Subid,
			},
			Grouplabel: chunk.Grouplabel,
			Etimes:     marshalTDSMapToTrainerMap(chunk.Etimes),
			Values:     marshalTDSMapToTrainerMap(chunk.Values),
		})
		if errSend != nil {
			logr.WithError(errSend).Errorf("cannot send trained model log")
			return fmt.Errorf("cannot send trained model log: %v", err)
		}
	}
	return nil
}

func (s *trainerService) validateRequest(log *logrus.Entry, req *grpc_trainer_v2.CreateRequest) error {
	if req.UserId == "" {
		return s.failCreateRequest("UserId is nil", req, log)
	}

	// validate model definition object

	m := req.ModelDefinition
	if m == nil {
		return s.failCreateRequest("Model definition is not set", req, log)
	}
	if m.Name == "" {
		return s.failCreateRequest("Model definition name is not set", req, log)
	}
	if m.Framework == nil {
		return s.failCreateRequest("Framework is not set", req, log)
	}
	if m.Framework.Name == "" {
		return s.failCreateRequest("Framework name is not set", req, log)
	}
	if m.Framework.Version == "" {
		return s.failCreateRequest("Framework version is not set", req, log)
	}
	if ok, msg := validateFrameworks(m.Framework); !ok {
		return s.failCreateRequest(msg, req, log)
	}
	if len(m.Content) == 0 {
		return s.failCreateRequest("Model definition content is not set", req, log)
	}

	// validate Training object

	t := req.Training
	if t == nil {
		return s.failCreateRequest("Training is not set", req, log)
	}
	if t.Command == "" {
		return s.failCreateRequest("Training command is not set", req, log)
	}
	if t.InputData == nil || len(t.InputData) == 0 {
		return s.failCreateRequest("Training input data is not set", req, log)
	}
	if len(t.InputData) > 1 {
		return s.failCreateRequest("Training input data can only contain one id", req, log)
	}
	if t.OutputData != nil && len(t.OutputData) > 1 {
		return s.failCreateRequest("Training output data can only contain one id", req, log)
	}

	// validate datastores

	ds := req.Datastores
	if ds == nil {
		return s.failCreateRequest("Data stores is not set", req, log)
	}
	if len(ds) == 0 {
		return s.failCreateRequest("Data stores is empty", req, log)
	}

	for _, name := range t.InputData {
		ds := findDatastore(name, req.Datastores)
		if ds == nil {
			return s.failCreateRequest(fmt.Sprintf("Training input data reference '%s' does not reference an existing datastore id.", name), req, log)
		}
		if err := s.validateDatastore(ds, req, log); err != nil {
			return err
		}
	}

	if len(t.OutputData) > 0 {
		for _, name := range t.OutputData {
			ds := findDatastore(name, req.Datastores)
			if ds == nil {
				return s.failCreateRequest(fmt.Sprintf("Training output data reference '%s' does not reference an existing datastore id.", name), req, log)
			}
			if err := s.validateDatastore(ds, req, log); err != nil {
				return err
			}
		}
	}
	return nil
}

func findDatastore(id string, ds []*grpc_trainer_v2.Datastore) *grpc_trainer_v2.Datastore {
	for _, v := range ds {
		if v.Id == id {
			return v
		}
	}
	return nil
}

func (s *trainerService) failCreateRequest(msg string, req *grpc_trainer_v2.CreateRequest, log *logrus.Entry) error {
	return s.failCreateRequestWithCode(trainerClient.ErrInvalidManifestFile, msg, req, log)
}

func (s *trainerService) failCreateRequestWithCode(errorCode string, msg string, req *grpc_trainer_v2.CreateRequest, log *logrus.Entry) error {
	log.Errorf("Failed to validate CreateRequest: %s", msg)

	// send error event as monitoring metric
	counter := s.metrics.trainingJobFailedCounter.With("type", "client", "errorcode", errorCode)
	if req.ModelDefinition != nil && req.ModelDefinition.Framework != nil {
		counter = counter.With("framework", req.ModelDefinition.Framework.Name, "version", req.ModelDefinition.Framework.Version)
	}
	if req.Training != nil && req.Training.Resources != nil {
		counter = counter.With("gpus", strconv.Itoa(int(req.Training.Resources.Gpus)),
			"cpus", strconv.Itoa(int(req.Training.Resources.Cpus)),
			"memory", strconv.Itoa(int(req.Training.Resources.Memory)))
	}
	counter.Add(1)

	return gerrf(codes.InvalidArgument, msg)
}

func (s *trainerService) validateDatastore(ds *grpc_trainer_v2.Datastore, req *grpc_trainer_v2.CreateRequest, log *logrus.Entry) error {

	if ds == nil {
		return s.failCreateRequest("Data store is not set", req, log)
	}
	if ds.Id == "" {
		return s.failCreateRequest("Data store id is not set", req, log)
	}
	if ds.Connection == nil || len(ds.Connection) == 0 {
		return s.failCreateRequest("Data store connection info not set", req, log)
	}
	if ds.Fields == nil || len(ds.Fields) == 0 || ds.Fields["bucket"] == "" {
		return s.failCreateRequest("Data store bucket is not set", req, log)
	}

	ostore, err := storage.CreateDataStore(ds.Type, ds.Connection)
	if err != nil {
		log.Errorf("Validation failed: %s", err.Error())
		return s.failCreateRequestWithCode(trainerClient.ErrInvalidCredentials,
			fmt.Sprintf("Data store authentication information for id '%s' incorrect or there is a connection problem", ds.Id), req, log)
	}

	if err := ostore.Connect(); err != nil {
		log.Errorf("Validation failed: %s", err.Error())
		return s.failCreateRequestWithCode(trainerClient.ErrInvalidCredentials,
			fmt.Sprintf("Data store authentication information for id '%s' incorrect or there is a connection problem", ds.Id), req, log)
	}

	// validate bucket (or container as it is called in Swift)
	bucket := ds.Fields["bucket"]
	if bucket != "" {
		exists, err := ostore.ContainerExists(bucket)
		if !exists || err != nil {
			return s.failCreateRequestWithCode(trainerClient.ErrInvalidCredentials,
				fmt.Sprintf("Data store bucket '%s' for data store id '%s' incorrect or there is a connection problem", bucket, ds.Id), req, log)
		}
	}
	return nil
}

// lcmClient established a connection if the trainerService has nothing existing cached
func (s *trainerService) lcmClient() (client.LcmClient, error) {
	if s.lcm == nil {
		return client.NewLcm(nil)
	}
	return s.lcm, nil
}

func (s *trainerService) tdsClient() (tdsClient.TrainingDataClient, error) {
	if s.tds == nil {
		address := fmt.Sprintf("ffdl-trainingdata.%s.svc.cluster.local:80", config.GetPodNamespace())
		tds, err := tdsClient.NewTrainingDataClientWithAddress(address)
		if err != nil {
			return nil, err
		}
		s.tds = tds
	}
	return s.tds, nil
}

func createJobConfig(trainingID string, req *grpc_trainer_v2.CreateRequest, trainingData *grpc_trainer_v2.Datastore,
	trainingResults *grpc_trainer_v2.Datastore) (*service.JobDeploymentRequest, error) {
	logr := logger.LocLogger(logWith(trainingID, req.UserId))

	// Environment variables passed in
	envvars := make(map[string]string)

	// Fetching data from user's Object Store with following info
	envvars["DATA_STORE_TYPE"] = trainingData.Type
	envvars["DATA_STORE_AUTHURL"] = trainingData.Connection["auth_url"]
	if trainingData.Connection["project_id"] != "" {
		envvars["DATA_STORE_PROJECTID"] = trainingData.Connection["project_id"]
	}
	if trainingData.Connection["type"] != "" {
		envvars["DATA_STORE_TYPE"] = trainingData.Connection["type"]
	}
	if trainingData.Connection["user_name"] != "" {
		envvars["DATA_STORE_USERNAME"] = trainingData.Connection["user_name"]
	}
	if trainingData.Connection["password"] != "" {
		envvars["DATA_STORE_APIKEY"] = trainingData.Connection["password"]
	}
	if trainingData.Connection["domain_name"] != "" {
		envvars["DATA_STORE_DOMAINNAME"] = trainingData.Connection["domain_name"]
	}
	if trainingData.Connection["region"] != "" {
		envvars["DATA_STORE_REGION"] = trainingData.Connection["region"]
	}
	envvars["DATA_STORE_OBJECTID"] = trainingData.Fields["bucket"]

	// Allow to fetch model from DLaaS's Object Store to the container
	osConf := config.GetDataStoreConfig()
	envvars["MODEL_STORE_USERNAME"] = osConf[storage.UsernameKey]
	envvars["MODEL_STORE_APIKEY"] = osConf[storage.PasswordKey]
	envvars["MODEL_STORE_AUTHURL"] = osConf[storage.AuthURLKey] // this will inside SL so we need the internal one
	if osConf[storage.StorageType] != "" {
		envvars["MODEL_STORE_TYPE"] = osConf[storage.StorageType]
	}
	// only needed for Bluemix objectstore
	if val, ok := osConf[storage.DomainKey]; ok {
		envvars["MODEL_STORE_DOMAINNAME"] = val
	}
	if val, ok := osConf[storage.RegionKey]; ok {
		envvars["MODEL_STORE_REGION"] = val
	}
	if val, ok := osConf[storage.DomainKey]; ok {
		envvars["MODEL_STORE_PROJECTID"] = val
	}
	envvars["MODEL_STORE_OBJECTID"] = req.ModelDefinition.Location

	// "Storing trained model in DLaaS Object Store with following info:"
	envvars["RESULT_STORE_TYPE"] = trainingResults.Type
	envvars["RESULT_STORE_USERNAME"] = trainingResults.Connection["user_name"]
	envvars["RESULT_STORE_APIKEY"] = trainingResults.Connection["password"]
	envvars["RESULT_STORE_AUTHURL"] = trainingResults.Connection["auth_url"]
	if trainingResults.Connection[storage.StorageType] != "" {
		envvars["RESULT_STORE_TYPE"] = trainingResults.Connection[storage.StorageType]
	}
	// only needed for Bluemix objectstore
	if trainingResults.Connection["domain_name"] != "" {
		envvars["RESULT_STORE_DOMAINNAME"] = trainingResults.Connection["domain_name"]
	}
	if trainingResults.Connection["region"] != "" {
		envvars["RESULT_STORE_REGION"] = trainingResults.Connection["region"]
	}
	if trainingResults.Connection["project_id"] != "" {
		envvars["RESULT_STORE_PROJECTID"] = trainingResults.Connection["project_id"]
	}
	envvars["RESULT_STORE_OBJECTID"] = fmt.Sprintf("%s/%s", trainingResults.Fields["bucket"], trainingID)

	// Storing data in container at
	envvars["DATA_DIR"] = trainingData.Fields["bucket"]

	// Storing model in container at
	envvars["MODEL_DIR"] = "/model-code"

	// Storing trained model at
	envvars["RESULT_DIR"] = trainingResults.Fields["bucket"]

	// TODO: This is pointing to currently where the logs are put, but should be redefined per nfs log mount proposal.
	// (by the time it gets to the learners/log-collectors, it will be "/job/logs", at the time of this writing.)
	envvars["LOG_DIR"] = "/logs"

	re := regexp.MustCompile(`\r?\n`)
	input := re.ReplaceAllString(fmt.Sprint(req.Training.Command), " ")

	envvars["TRAINING_COMMAND"] = input

	envvars["TRAINING_ID"] = trainingID

	envvars["GPU_COUNT"] = strconv.FormatFloat(float64(req.Training.Resources.Gpus), 'f', 6, 64)

	envvars["SCHED_POLICY"] = strings.ToLower(req.Training.Resources.Schedpolicy)

	// tag to use to lookup learner image to use; this is a Docker image tag
	if req.ModelDefinition.Framework.ImageTag != "" {
		envvars["DLAAS_LEARNER_IMAGE_TAG"] = req.ModelDefinition.Framework.ImageTag
	}

	// envvar for profile
	if req.Training.Profiling {
		envvars["DLAAS_PROFILING"] = "true"
	}

	// labels
	labels := make(map[string]string)
	labels["training_id"] = trainingID
	labels["user_id"] = req.UserId

	u4, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	logr.Debugf("Training job env vars: %v", envvars)

	job := &service.JobDeploymentRequest{
		Name:       u4.String(),
		Resources:  getResourceRequirements(req.Training),
		EnvVars:    envvars,
		Labels:     labels,
		UserId:     req.UserId,
		TrainingId: trainingID,
		Framework:  req.ModelDefinition.Framework.Name,
		Version:    req.ModelDefinition.Framework.Version,
	}
	if req.EvaluationMetrics != nil {
		logr.Debugf("EMExtractionSpec ImageTag: %s", req.EvaluationMetrics.ImageTag)
		wrapper := make(map[string]interface{})
		wrapper["evaluation_metrics"] = req.EvaluationMetrics
		data, err := yaml.Marshal(wrapper)
		if err != nil {
			logr.WithError(err).Errorf("Can't re-marshal evaluation metrics specification")
		}
		job.EvaluationMetricsSpec = string(data)
		logr.Debugf("Set evaluation_metrics to: %s<eof>", job.EvaluationMetricsSpec)
	}

	return job, nil
}

func setDefaultResourceRequirements(t *grpc_trainer_v2.Training) {
	if t == nil || t.Resources == nil {
		t.Resources = &grpc_trainer_v2.ResourceRequirements{ // set sensible defaults
			Cpus:        5.0,
			Gpus:        1.0,
			Memory:      12.0,
			MemoryUnit:  grpc_trainer_v2.SizeUnit_GiB,
			Learners:    1,
			Schedpolicy: "dense",
		}
		return
	}
	if t.Resources.Cpus == 0 {
		t.Resources.Cpus = 5.0
	}
	if t.Resources.Memory == 0 {
		t.Resources.Memory = 12
		t.Resources.MemoryUnit = grpc_trainer_v2.SizeUnit_GiB
	}
	if t.Resources.Schedpolicy == "" || strings.ToLower(t.Resources.Schedpolicy) != "spread" {
		t.Resources.Schedpolicy = "dense"
	}
}

func getResourceRequirements(t *grpc_trainer_v2.Training) *service.ResourceRequirements {
	return &service.ResourceRequirements{
		Cpus:        float64(t.Resources.Cpus),
		Gpus:        float64(t.Resources.Gpus),
		Memory:      float64(t.Resources.Memory),
		MemoryUnit:  service.ResourceRequirements_MemoryUnit(service.ResourceRequirements_MemoryUnit_value[t.Resources.MemoryUnit.String()]),
		Storage:     float64(t.Resources.Storage),
		StorageUnit: service.ResourceRequirements_MemoryUnit(service.ResourceRequirements_MemoryUnit_value[t.Resources.StorageUnit.String()]),
		Learners:    t.Resources.Learners,
	}
}

// getOutputDatastore retrieves the output data store or return the internal datastore if none has been defined
func (s *trainerService) getOutputDatastore(outputData []string, datastores []*grpc_trainer_v2.Datastore) *grpc_trainer_v2.Datastore {
	var ds *grpc_trainer_v2.Datastore
	if len(outputData) > 0 {
		ds = findDatastore(outputData[0], datastores) // we assume there is only one output data at this point b/c the underlying system does not support more
	}
	if ds == nil {
		ds = &grpc_trainer_v2.Datastore{
			Id:         internalObjectStoreID,
			Type:       config.GetDataStoreType(),
			Connection: config.GetDataStoreConfig(),
			Fields:     map[string]string{"bucket": s.trainedModelsBucket},
		}
	}
	return ds
}

// getOutputStoreForService is a wrapper function to make the logic in trainerService.getOutputDatastore available for testing
func getOutputStoreForService(s *trainerService, outputData []string, datastores []*grpc_trainer_v2.Datastore) *grpc_trainer_v2.Datastore {
	return s.getOutputDatastore(outputData, datastores)
}

func getModelsBucket() string {
	if viper.IsSet(modelsBucketKey) {
		return viper.GetString(modelsBucketKey)
	}
	return defaultModelsBucket
}

func getTrainedModelsBucket() string {
	if viper.IsSet(trainedModelsBucketKey) {
		return viper.GetString(trainedModelsBucketKey)
	}
	return defaultTrainedModelsBucket
}

func getModelZipFileName(trainingID string) string {
	return fmt.Sprintf("%s.zip", trainingID)
}

func rateLimitTrainingJob(records []*TrainingRecord, limit int, logr *logger.LocLoggingEntry) bool {
	var rateLimit = false

	var totalGPUsUsedCount float32
	var matchingGPUConsumingRecords []*TrainingRecord
	for _, record := range records {
		trainingStatus := record.TrainingStatus.Status
		if trainingStatus == grpc_trainer_v2.Status_COMPLETED || trainingStatus == grpc_trainer_v2.Status_HALTED || trainingStatus == grpc_trainer_v2.Status_FAILED {
			//ignore these since they don't count towards active resource usage
		} else {
			matchingGPUConsumingRecords = append(matchingGPUConsumingRecords, record)
			totalGPUsUsedCount = totalGPUsUsedCount + record.Training.Resources.Gpus
		}
	}

	if int(totalGPUsUsedCount) > limit {
		rateLimit = true
		logr.Infof("current logger level is %v and the check is %v", logr.Logger.Level, logr.Logger.Level <= logrus.DebugLevel)
		if logr.Logger.Level >= logrus.DebugLevel {
			for _, record := range matchingGPUConsumingRecords {
				logr.Debugf("Found a gpu consuming training %v has a status %s and using gpus %v with submission time as %v and process start time as %v and error code %v",
					record.TrainingID, record.TrainingStatus.Status, record.Training.Resources.Gpus, record.TrainingStatus.SubmissionTimestamp,
					record.TrainingStatus.ProcessStartTimestamp, record.TrainingStatus.ErrorCode)
			}
		}

	}
	logr.Infof("Found %d matching records out of %d searched and a total of %v gpus are in use", len(matchingGPUConsumingRecords), len(records), totalGPUsUsedCount)

	return rateLimit
}
