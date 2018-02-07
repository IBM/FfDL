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
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	k8srest "k8s.io/client-go/rest"

	"github.ibm.com/ffdl/ffdl-core/commons/config"

	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	trainerClient "github.ibm.com/ffdl/ffdl-core/trainer/client"

	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
)

//GetTrainerStatus gets Trainer Informations
func (s *lcmService) GetTrainerStatus(ctxOrig context.Context, tclient trainerClient.TrainerClient, req *service.TrainerContainerInfosRequest) (grpc_trainer_v2.Status, error) {
	logr := logger.LocLogger(s.logWithTrainerContainerInfosRequest(req))

	var Status grpc_trainer_v2.Status
	ctx, _ := context.WithTimeout(context.Background(), time.Second*60)

	resp, err := tclient.Client().GetTrainingStatusID(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})
	if err != nil {
		logr.WithError(err).Debugf("Trainer readOne service call failed, will retry")
	}

	if resp != nil {
		Status = resp.Status
		logr.Debugf("GetTrainingStatusID success: %d", Status)
	}

	return Status, nil
}

func (s *lcmService) GetTrainerStatusID(tclient trainerClient.TrainerClient, req *service.TrainerContainerInfosRequest) (grpc_trainer_v2.Status, error) {
	logr := logger.LocLogger(s.logWithTrainerContainerInfosRequest(req))

	var Status grpc_trainer_v2.Status
	ctx, _ := context.WithTimeout(context.Background(), time.Second*60)

	resp, err := tclient.Client().GetTrainingStatusID(ctx, &grpc_trainer_v2.GetRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
	})
	if err != nil {
		logr.WithError(err).Debugf("Trainer readOne service call failed, will retry")
	}

	if resp != nil {
		Status = resp.Status
	}
	return Status, err
}

// GetPodName gets the pods associated with the training ID
func (s *lcmService) GetPodsForTrainingID(logr *logger.LocLoggingEntry, trainingID string) (*v1core.PodList, error) {
	podInterface := s.k8sClient.Core().Pods(config.GetLearnerNamespace())
	podIDSelector := "training_id==" + trainingID
	podList, err := podInterface.List(metav1.ListOptions{LabelSelector: podIDSelector})
	if err != nil {
		logr.Errorf("podInterface.List(...) returned error: %s", err)
		return nil, err
	}
	logr.Debugf("podList.Item count: %d", len(podList.Items))
	return podList, nil
}

func (s *lcmService) getTrainingLogStreamFromObjStore(trainerClient trainerClient.TrainerClient,
	req *service.TrainerContainerInfosRequest,
	outStream service.LifecycleManager_GetTrainingLogStreamServer) error {

	logrr := s.logWithTrainerContainerInfosRequest(req)
	logrr.Info("Entry")
	logr := logger.LocLoggerCategorized(logrr, logger.LogCategoryGetTrainingLogStreamFromObjStore)

	ctx, _ := context.WithTimeout(outStream.Context(), time.Minute*3)
	stream, err := trainerClient.Client().GetTrainedModelLogsFromObjStore(ctx, &grpc_trainer_v2.TrainedModelLogRequest{
		TrainingId: req.TrainingId,
		UserId:     req.UserId,
		IsMetrics:  req.Metrics,
		IsSummary:  req.Summary,
	})
	if err != nil {
		logr.WithError(err).Errorf("Training job not fetched.")
		return sendErrorMsg(logr, outStream, true,
			fmt.Errorf("Training job not fetched: %v", err))
	}
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logr.WithError(err).Errorf("Cannot read trained model log.")
			return sendErrorMsg(logr, outStream, true,
				fmt.Errorf("Cannot read trained model log: %v", err))
		}
		errSend := outStream.Send(&service.TrainerLogStreamResponse{
			Data: chunk.Data,
		})
		if errSend != nil {
			logr.WithError(errSend).Errorf("Cannot read trained model log.")
			return sendErrorMsg(logr, outStream, false,
				fmt.Errorf("outStream.Send(...) returned error: %v", errSend))
		}
		time.Sleep(time.Second * 1)
	}
	return nil
}

func (s *lcmService) checkPersistentStoreStatus(ctx context.Context,
	req *service.TrainerContainerInfosRequest,
	outStream service.LifecycleManager_GetTrainingLogStreamServer,
	trainer trainerClient.TrainerClient,
	logr *logger.LocLoggingEntry) (bool, error) {

	logr.Debug("checkPersistentStoreStatus(...) enter")

	// loop in case of STORING state, we might as well wait.
	for i := 0; i < 10; i++ {
		persistentStatus, errTrainerStatus := s.GetTrainerStatus(ctx, trainer, req)
		if errTrainerStatus != nil {
			logr.WithError(errTrainerStatus).Debugf("GetTrainerStatus returned error, might be ok")
			return false, errTrainerStatus
		}
		shouldBreak := false
		switch persistentStatus {
		case grpc_trainer_v2.Status_HALTED, grpc_trainer_v2.Status_FAILED, grpc_trainer_v2.Status_COMPLETED:
			logr.Debugf("Job is done, db status is: %s, fetching from obj store",
				persistentStatus.String())
			return true, s.getTrainingLogStreamFromObjStore(trainer, req, outStream)
		case grpc_trainer_v2.Status_STORING:
			logr.Debugf("Waiting for DB to transition from STORING")
			time.Sleep(time.Second * 2)
			// go has implicit break
		default:
			shouldBreak = true
		}
		if shouldBreak {
			// break from loop
			break
		}
	}
	logr.Debug("checkPersistentStoreStatus(...) exit")
	return false, nil
}

// This gets called when eof is found when reading the logs
func (s *lcmService) isJobPrematurelyRestarting(logr *logger.LocLoggingEntry, podName string,
	req *service.TrainerContainerInfosRequest, shouldConsiderStoringDone bool) bool {

	logr.Debug("Pause for a second to give time for etcd to update")
	time.Sleep(time.Second * 1)
	status, isComplete := isJobCompleted(logr, s, req, shouldConsiderStoringDone)
	if isComplete == false {
		return false
	}
	// Seems to be tricky race conditions here?
	if shouldConsiderStoringDone && status == grpc_trainer_v2.Status_PROCESSING {
		// Really?  Or is it just being really slow to update?
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second * 2)
			_, isComplete := isJobCompleted(logr, s, req, shouldConsiderStoringDone)
			if isComplete == false {
				return false
			}
		}

		podInterface := s.k8sClient.Core().Pods(config.GetLearnerNamespace())
		logr.Debug("asking for pod")
		for i := 0; i < 1; i++ {
			pod, podGetError := podInterface.Get(podName, metav1.GetOptions{})

			logr.Debug("back from asking for pod")
			if podGetError != nil {
				// must be gone?
				// One more try with etcd, for the fun of it.
				_, isComplete := isJobCompleted(logr, s, req, shouldConsiderStoringDone)
				if isComplete == false {
					return false
				}
				logr.WithError(podGetError).Errorf("Can not get POD, assume it's crashed")
				return true
			}
			pod, podGetError = podInterface.UpdateStatus(pod)
			logr.Debug("back from asking UpdateStatus")
			if podGetError != nil {
				// must be gone?
				_, isComplete := isJobCompleted(logr, s, req, shouldConsiderStoringDone)
				if isComplete == false {
					return false
				}
				logr.WithError(podGetError).Error("Can not UpdateStatus, assume it's crashed")
				return true
			}
			if pod != nil {
				logr.Debugf("Pod status: phase %s, reason: %s, message: %s", pod.Status.Phase,
					pod.Status.Reason, pod.Status.Message)
				if pod.Status.Phase != v1core.PodRunning && pod.Status.Phase != v1core.PodSucceeded {
					return true
				}
			} else {
				return true
			}
		}
		return false
	}
	return true
}

// waitForLoggingPods will wait for the job to be ready for logging, and will return an appropriate ReadCloser,
// or it will download the logs from the object store, and send them, or it will send an error message.
// Besides being called in normal conditions, this is also meant to be called in the case of a pod
// restart.
func (s *lcmService) waitForLoggingPods(logr *logger.LocLoggingEntry, req *service.TrainerContainerInfosRequest,
	outStream service.LifecycleManager_GetTrainingLogStreamServer) (string, io.ReadCloser, error) {

	logr.Info("Entry")
	var podName string

	trainer, err := trainerClient.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("Cannot create client for trainer service.")
		return podName, nil, sendErrorMsg(logr, outStream, true,
			fmt.Errorf("Cannot create client for trainer service: %s", err))
	}
	defer trainer.Close()

	logr.Info("Creating timeout context")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	var podInterface v1.PodInterface
	var pods *v1core.PodList
	// Loop if needed until the job is started.  But don't wait too long... if the job doesn't start soon
	// assume something is wrong, and then the user will have to retry, I guess.
	reportInterval := (time.Second * 5)
	var timeOfLastReport time.Time
	if req.Metrics {
		// Don't status report for metrics
		// timeOfLastReport = time.Now().Add(time.Hour * 2000)

		timeOfLastReport = time.Now().Add(-reportInterval)
	} else {
		timeOfLastReport = time.Now().Add(-reportInterval)
	}
	jobStatusFromZK := "Not Started"
	var numberRetries = 1
	if req.Follow {
		// If following, we'll keep retrying for a while
		numberRetries = 40
	}

	for i := 0; i < numberRetries; i++ {
		jobStatus, errZK := s.GetTrainerStatusID(trainer, req)

		if errZK != nil || jobStatus == 0 {
			jobIsDone, err := s.checkPersistentStoreStatus(ctx, req, outStream, trainer, logr)
			if jobIsDone {
				return podName, nil, err
			}
			if req.Summary && req.Metrics {
				// time.Sleep(time.Second * 1)
			} else {
				// This is a somewhat questionable message to send to the user, but, hey, it's truthful.
				durationSinceLastReport := time.Now().Sub(timeOfLastReport)
				if durationSinceLastReport > reportInterval {
					logr.Debugf("durationSinceLastReport: %d", durationSinceLastReport)
					sendStatusMessage(logr, outStream, "Unknown", req.Metrics)
					timeOfLastReport = time.Now()
				}
				time.Sleep(time.Second * 3)
			}

		} else {
			jobStatus, errZK := s.GetTrainerStatusID(trainer, req)

			if errZK != nil || jobStatus == 0 {
				jobIsDone, err2 := s.checkPersistentStoreStatus(ctx, req, outStream, trainer, logr)
				if jobIsDone {
					return podName, nil, err2
				}
				logr.WithError(err2).Errorf("checkPersistentStoreStatus returned error")
				return podName, nil, err2
			}

			// Go ahead and report the status to the user at this point
			durationSinceLastReport := time.Now().Sub(timeOfLastReport)
			if durationSinceLastReport > reportInterval {
				logr.Debugf("durationSinceLastReport: %d", durationSinceLastReport)
				sendStatusMessage(logr, outStream, jobStatus.String(), req.Metrics)
				timeOfLastReport = time.Now()
			}

			if isJobDone(jobStatus.String(), logr) {
				logr.Debugf("Job completed, fetching from obj store")
				return podName, nil, s.getTrainingLogStreamFromObjStore(trainer, req, outStream)
			}
			selector := "training_id==" + req.TrainingId

			podInterface = s.k8sClient.Core().Pods(config.GetLearnerNamespace())
			pods, err = podInterface.List(metav1.ListOptions{LabelSelector: selector})
			if err != nil {
				logr.WithError(err).Debugf("podInterface.List returned maybe ok error")
				continue
			}
			logr.Debugf("podList.Item count: %d, job status: %s", len(pods.Items), jobStatus)
			if jobStatus != grpc_trainer_v2.Status_NOT_STARTED {
				if pods == nil || pods.Size() == 0 {
					logr.Debugf("zk/getStatus, but no pods yet (?): %s",
						jobStatus.String())
					time.Sleep(time.Second * 2)
					continue
				}
				if pods != nil && pods.Size() > 0 {
					podName = getLearnerPodName(pods, logr, learnerContainerName)
					logr.Debugf("GetPodsForTrainingID returned : %s, follow is %t",
						podName, req.Follow)

					podInterface = s.k8sClient.Core().Pods(config.GetLearnerNamespace())
				}
				break
			}

		}
		time.Sleep(time.Second * 2)
	}

	if pods == nil || pods.Size() == 0 {
		// Shrug.  Try again with the object store.
		logr.Debugf("No pods found, calling getTrainingLogStreamFromObjStore")
		return podName, nil, s.getTrainingLogStreamFromObjStore(trainer, req, outStream)
	}

	var readCloser io.ReadCloser

	// Make sure the logging container is ready, or else we'll have problems.

	logr.Debugf("Make sure the logging container is ready, or else we'll have problems.")
	var restReq *k8srest.Request
	for i := 0; i < 64; i++ {
		jobStatus, errZK := s.GetTrainerStatusID(trainer, req)
		if errZK != nil || jobStatus == 0 {
			logr.Debugf("Checking the persistent store status")
			jobIsDone, err2 := s.checkPersistentStoreStatus(ctx, req, outStream, trainer, logr)
			if jobIsDone {
				logr.Debugf("Persistent store reports job done")
				return podName, nil, err2
			}

			logr.WithError(errZK).Errorf("ZooKeeper returned error")
			return podName, nil, errZK
		}

		logr.Debugf("zk status: %s", jobStatus)
		if isJobDone(jobStatus.String(), logr) {
			logr.Debugf("Job is done, db status is: %s, fetching from obj store", jobStatus)
			return podName, nil, s.getTrainingLogStreamFromObjStore(trainer, req, outStream)
		}

		// Go ahead and report the status to the user at this point
		durationSinceLastReport := time.Now().Sub(timeOfLastReport)
		if durationSinceLastReport > reportInterval {
			logr.Debugf("durationSinceLastReport: %d", durationSinceLastReport)
			sendStatusMessage(logr, outStream,
				jobStatusFromZK,
				req.Metrics)
			timeOfLastReport = time.Now()
		}

		var learnerOrLogCollectorLabel string
		if req.Metrics {
			learnerOrLogCollectorLabel = logCollectorContainerName
		} else {
			learnerOrLogCollectorLabel = learnerContainerName
		}

		// Should be able to call GetContainerStatus(...), but the k8s interface seems to have some issues. -sb
		if isLogCollectorReady(pods, logr, learnerOrLogCollectorLabel) {
			logr.Debugf("logging container is ready! podname: %s", podName)
			for i := 0; i < 10; i++ {
				restReq = podInterface.GetLogs(podName, &v1core.PodLogOptions{
					Follow:    req.Follow,
					Container: learnerOrLogCollectorLabel,
				})
				var err error
				logr.Debugf("Calling restReq.Stream()")
				readCloser, err = restReq.Stream()
				logr.Debugf("Back from restReq.Stream()")
				if err == nil && readCloser != nil {
					break
				} else {
					// Not sure yet if this should be a real error. -sb
					// Getting a "resource name may not be empty" error here
					if err != nil {
						logr.WithError(err).Debugf(
							"restReq.Stream() error (might be ok, try again: %d)", i)
					}
					// return podName, nil, err
					time.Sleep(time.Second * 2)
					continue
				}
			}
			if readCloser == nil {
				logr.Debugf("restReq.Stream() returned nil after multiple tries")
				continue
			}
			logr.Debugf("Found logger, obtained log stream!!")
			break
		} else {
			secondsToSleep := 3
			logr.Debugf("sleeping for %d seconds waiting for logging containter", secondsToSleep)
			time.Sleep(time.Second * time.Duration(secondsToSleep))
		}
	}

	durationSinceLastReport := time.Now().Sub(timeOfLastReport)
	if durationSinceLastReport > reportInterval {
		logr.Debugf("durationSinceLastReport: %d", durationSinceLastReport)
		sendStatusMessage(logr, outStream, jobStatusFromZK, req.Metrics)
		timeOfLastReport = time.Now()
	}

	if readCloser == nil {
		// Shrug.  Try again with the object store.
		logr.Debugf("restReq.Stream() returned nil readCloser, fetching from obj store")
		return podName, nil, s.getTrainingLogStreamFromObjStore(trainer, req, outStream)
	}
	logr.Debugf("Returning with hopefully valid readCloser and no error")
	return podName, readCloser, nil
}

// GetTrainingLogStream returns a list of docker container IDs along with their hosts.
func (s *lcmService) GetTrainingLogStream(req *service.TrainerContainerInfosRequest,
	outStream service.LifecycleManager_GetTrainingLogStreamServer) error {

	logrr := s.logWithTrainerContainerInfosRequest(req)
	logrr.Info("Entry")
	// Note: you won't see logs for this function past this unless you set DLAAS_LOG_GETTRAININGLOGSTREAM=true
	logr := logger.LocLoggerCategorized(logrr, logger.LogCategoryGetTrainingLogStream)

	podName, readCloser, err := s.waitForLoggingPods(logr, req, outStream)

	if err != nil {
		logr.WithError(err).Debug("waitForLoggingPods returned error")
		return err
	}
	if readCloser == nil {
		logr.Print("Returning with nil, assume object store download")
		return nil
	}

	defer readCloser.Close()

	bufSize := 1024 * 2
	// bufSize := 256
	buf := make([]byte, 0, bufSize)

	totalRead := 0
	retries := 0
	diagnoseChunkBoundaries := false
	for {
		logr.Debug("Calling read")
		n, err := readCloser.Read(buf[:cap(buf)])
		logr.Debugf("readCloser returned %d bytes", n)

		if n > 0 {
			logr.Debugf("readCloser returned %d bytes", n)
			if diagnoseChunkBoundaries {
				errSendx := outStream.Send(&service.TrainerLogStreamResponse{
					Data: []byte("[[["),
				})
				if errSendx != nil {
					logr.WithError(errSendx).Error("outStream.Send(...) returned error")
					return errSendx
				}
			}

			errSend := outStream.Send(&service.TrainerLogStreamResponse{
				Data: buf[:n],
			})
			if errSend != nil {
				logr.WithError(errSend).Error("outStream.Send(...) returned error")
				return errSend
			}
			if diagnoseChunkBoundaries {
				errSendx := outStream.Send(&service.TrainerLogStreamResponse{
					Data: []byte("]]]"),
				})
				if errSendx != nil {
					logr.WithError(errSendx).Error("outStream.Send(...) returned error")
					return errSendx
				}
			}
		}
		totalRead += n
		if err == io.EOF {
			logr.WithError(err).Debug("eof returned from readCloser.Read(...)")
			if n == 0 && totalRead == 0 {
				// How often, if ever, does this condition occur?
				logr.Debugf(
					"Found EOF, didn't get any bytes, snoozing a bit, then check again.")
				time.Sleep(time.Second * 2)

				if retries < 10 {
					retries++
					podName, readCloser, err = s.waitForLoggingPods(logr, req, outStream)

					if err != nil {
						logr.WithError(err).Debug("waitForLoggingPods returned error")
						return err
					}
					if readCloser == nil {
						noLogReaderError := fmt.Errorf("Can't get log reader")
						logr.WithError(noLogReaderError).Debugf(
							"waitForLoggingPods(...) return nil")
						return noLogReaderError
					}
					// continue
				} else {
					return fmt.Errorf(
						"After multiple retries, can't read training logs for %s",
						req.TrainingId)
				}
			} else if n == 0 && s.isJobPrematurelyRestarting(logr,
				podName, req, true) {
				//
				logr.Debugf("Assume the job is restarting, restarting logging")

				sendStatusMessage(logr, outStream, "Restart", req.Metrics)

				time.Sleep(time.Second * 2)

				if retries < 10 {
					retries++
					podName, readCloser, err = s.waitForLoggingPods(logr, req, outStream)

					if err != nil {
						logr.WithError(err).Debug("waitForLoggingPods returned error")
						return err
					}
					if readCloser == nil {
						noLogReaderError := fmt.Errorf("Can't get log reader")
						logr.WithError(noLogReaderError).Debugf(
							"waitForLoggingPods(...) return nil")
						return noLogReaderError
					}
					continue
				} else {
					return fmt.Errorf(
						"After multiple retries, can't read training logs for %s",
						req.TrainingId)
				}
			}

			logr.WithError(err).Debugf("readCloser.Read(...) returned eof")
			break
		}
		if err != nil {
			logr.WithError(err).Errorf("readCloser.Read(...) returned error")
			if retries < 10 {
				retries++
				podName, readCloser, err = s.waitForLoggingPods(logr, req, outStream)

				if err != nil {
					logr.WithError(err).Debug("waitForLoggingPods returned error")
					return err
				}
				if readCloser == nil {
					return nil
				}
				// continue
			} else {
				return err
			}
		}
	}

	logr.Debug("Exit")

	return nil
}
