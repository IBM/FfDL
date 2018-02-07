package jobmonitor

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/metrics/statsd"

	"github.com/cenkalti/backoff"
	"github.com/go-kit/kit/metrics"

	"github.com/coreos/etcd/clientv3"

	"google.golang.org/grpc"

	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"github.ibm.com/ffdl/ffdl-core/lcm/lcmconfig"

	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	"github.ibm.com/ffdl/ffdl-core/lcm/coord"

	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	lcmClient "github.ibm.com/ffdl/ffdl-core/commons/service/client"
	"github.ibm.com/ffdl/ffdl-core/trainer/client"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"
)

// Confuse `go vet' to not check this `Errorf' call. :(
// See https://github.com/grpc/grpc-go/issues/90
var gerrf = grpc.Errorf

const (
	zkLearners = "learners"
	zkLearner  = "learner_"
	zkStatus   = "status"
)

const (
	numRetries                    = 10
	insuffResourcesRetries        = 40
	ctxTimeout                    = 10 * time.Second
)

type jobMonitorMetrics struct {
	failedETCDConnectivityCounter, failedK8sConnectivityCounter, insufficientK8sResourcesErrorCounter, failedImagePullK8sErrorCounter,
	failedETCDWatchCounter metrics.Counter
}

//JobMonitor ...
type JobMonitor struct {
	k8sClient             kubernetes.Interface
	UseNativeDistribution bool
	TrainingID            string
	UserID                string
	JobName               string
	NumLearners           int
	trMap                 map[string]([]string)
	etcdClient            coord.Coordinator
	numTerminalLearners   uint64
	metrics               *jobMonitorMetrics
}

var failedTrainerConnectivityCounter metrics.Counter

// count etcd progress notifications (arrive every 10 mins)
var etcdJobProgressNotificationCounter uint32
var etcdLearnerProgressNotificationCounter uint32

// number of progress notifications to count before dropping a log line (e.g., 6 * 10 minutes = log every hour)
const etcdProgressNotificationLogFrequency = 6

//NewJobMonitor ...
func NewJobMonitor(trainingID string, userID string, numLearners int, jobName string, useNativeDistribution bool, statsdClient *statsd.Statsd, logr *logger.LocLoggingEntry) (*JobMonitor, error) {

	logr.Infof("Starting Job Monitor service for training %s", trainingID)
	// assert necessary config keys
	config.FatalOnAbsentKey(config.ETCDEndpoints)
	failedTrainerConnectivityCounter = statsdClient.NewCounter("jobmonitor.trainer.connectivity.failed", 1)

	jmMetrics := jobMonitorMetrics{
		failedETCDConnectivityCounter:        statsdClient.NewCounter("jobmonitor.etcd.connectivity.failed", 1),
		failedK8sConnectivityCounter:         statsdClient.NewCounter("jobmonitor.k8s.connectivity.failed", 1),
		insufficientK8sResourcesErrorCounter: statsdClient.NewCounter("jobmonitor.k8s.insufficientResources.failed", 1),
		failedImagePullK8sErrorCounter:       statsdClient.NewCounter("jobmonitor.k8s.imagePull.failed", 1),
		failedETCDWatchCounter:               statsdClient.NewCounter("jobmonitor.etcd.watch.failed", 1),
	}

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute

	var coordinator coord.Coordinator
	var err error
	err = backoff.RetryNotify(func() error {
		coordinator, err = coord.NewCoordinator(coord.Config{Endpoints: config.GetEtcdEndpoints(), Prefix: config.GetEtcdPrefix(),
			Cert: config.GetEtcdCertLocation(), Username: config.GetEtcdUsername(), Password: config.GetEtcdPassword()}, logr)
		return err
	}, defaultBackoff, func(err error, t time.Duration) {
		jmMetrics.failedETCDConnectivityCounter.Add(1)
	})

	//FIXME no defer close is being called here, so the etcdclient never closes
	if err != nil {
		logr.WithError(err).Errorf("Failed to connect to etcd while creating new lcm service for training %s", trainingID)

		if err := updateJobStatusOnError(trainingID, userID, client.ErrCodeEtcdConnection, service.StatusMessages_INTERNAL_ERROR.String(), logr); err != nil {
			logr.WithError(err).Errorf("Failed to write the status %s for training %s to trainer", grpc_trainer_v2.Status_FAILED, trainingID)
		}
		if err := KillDeployedJob(trainingID, userID, jobName, logr); err != nil {
			logr.WithError(err).Errorf("Failed to kill the deployed job %s", trainingID)
		}
		return nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(lcmconfig.GetKubernetesConfig())
	if err != nil {
		jmMetrics.failedK8sConnectivityCounter.Add(1)
		logr.WithError(err).Errorf("Failed to connect to k8s while creating new lcm service for training %s", trainingID)

		if err := updateJobStatusOnError(trainingID, userID, client.ErrCodeK8SConnection, service.StatusMessages_INTERNAL_ERROR.String(), logr); err != nil {
			logr.WithError(err).Errorf("Failed to write the status %s for training %s to trainer", grpc_trainer_v2.Status_FAILED, trainingID)
		}
		if err := KillDeployedJob(trainingID, userID, jobName, logr); err != nil {
			logr.WithError(err).Errorf("Failed to kill the deployed job %s", trainingID)
		}
		return nil, fmt.Errorf("Failed to connect to k8s")
	}

	jm := &JobMonitor{
		k8sClient:             k8sClient,
		UseNativeDistribution: useNativeDistribution,
		TrainingID:            trainingID,
		UserID:                userID,
		JobName:               jobName,
		NumLearners:           numLearners,
		trMap:                 initTransitionMap(),
		etcdClient:            coordinator,
		metrics:               &jmMetrics,
	}

	return jm, nil
}

//update job status in mongo
func updateJobStatusInTrainer(trainingID string, userID string, statusUpdate *client.TrainingStatusUpdate, logr *logger.LocLoggingEntry) error {
	updStatus := statusUpdate.Status
	logr.Infof("(updateJobStatus) Updating status of %s to %s", trainingID, updStatus.String())
	updateRequest := &grpc_trainer_v2.UpdateRequest{TrainingId: trainingID, Status: updStatus,
		UserId: userID, StatusMessage: statusUpdate.StatusMessage, ErrorCode: statusUpdate.ErrorCode}
	trainer, err := client.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("(updateJobStatus) Creating training client for status update failed. Training ID %s New Status %s", trainingID, updStatus.String())
	}
	defer trainer.Close()

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute
	defaultBackoff.MaxInterval = 5 * time.Second

	err = backoff.RetryNotify(func() error {
		_, err = trainer.Client().UpdateTrainingJob(context.Background(), updateRequest)
		return err
	}, defaultBackoff, func(err error, t time.Duration) {
		logr.WithError(err).Errorf("Failed to update status to the trainer. Retrying WARNING: Status updates for %s may be temporarily inconsistent due to failure to communicate with Trainer.", trainingID)
	})

	if err != nil {
		failedTrainerConnectivityCounter.Add(1)
		logr.WithError(err).Errorf("Failed to update status to the trainer. Already retried several times.WARNING : Status of job %s will likely be incorrect", trainingID)
		return err
	}

	if err := sendStatusUpdate(trainingID, updStatus.String(), logr); err != nil {
		logr.WithError(err).Errorf("Error when sending status update(s) for trainingID %s", trainingID)
	}

	return err
}

// update job status in mongo on error
func updateJobStatusOnError(trainingID string, userID string, errorCode string, statusMessage string, logr *logger.LocLoggingEntry) error {
	statusUpdate := client.TrainingStatusUpdate{
		Status: grpc_trainer_v2.Status_FAILED,
		Timestamp: float64(time.Now().UnixNano()) / 1000000000,
		ErrorCode: errorCode,
		StatusMessage: statusMessage,
	}
	return updateJobStatusInTrainer(trainingID, userID, &statusUpdate, logr)
}

//ManageDistributedJob ...manages a DLaaS training job
func (jm *JobMonitor) ManageDistributedJob(logr *logger.LocLoggingEntry) {
	go jm.checkIfJobStarted(logr)
	go jm.monitorLearnerForFailedImagePull(logr)
	go jm.monitorJob(logr)
}

//monitors the job at the path jobBasePath() generall /training_id/ under which there is /training_id/status/ indicating over all job status
//and there can be jobLearnerStatusPath() generally /training_id/learners/learner_1/status/ , 2 and 3 indicating status of individual learners
//the trailing slash on status/ on learner is important as it distinguishes the regex from status_summary_metrics
func (jm *JobMonitor) monitorJob(logr *logger.LocLoggingEntry) {

	type IndividualStatusChannelsWatch struct {
		key   string
		watch clientv3.WatchChan
	}

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute
	defaultBackoff.MaxInterval = 5 * time.Second
	err := backoff.RetryNotify(func() error {
		_, err := jm.etcdClient.PutIfKeyMissing(overallJobStatusPath(jm.TrainingID), grpc_trainer_v2.Status_NOT_STARTED.String(), logr)
		return err
	}, defaultBackoff, func(err error, t time.Duration) { jm.metrics.failedK8sConnectivityCounter.Add(1) })

	//FIXME should we kill the job here
	if err != nil {
		logr.WithError(err).Errorf("(monitorLearner)  Failed to set up the job status %s for the path %s :",
			grpc_trainer_v2.Status_NOT_STARTED.String(), overallJobStatusPath(jm.TrainingID))
	}

	done := make(chan string)
	overallJobStatusChannel := jm.etcdClient.WatchPath(context.Background(), overallJobStatusPath(jm.TrainingID), logr, clientv3.WithProgressNotify())
	var individualJobStatusChannels []*IndividualStatusChannelsWatch
	for i := 1; i <= jm.NumLearners; i++ {
		individualWatch := jm.etcdClient.WatchPath(context.Background(), indvidualJobStatusPath(jm.TrainingID, i), logr, clientv3.WithProgressNotify())
		item := IndividualStatusChannelsWatch{
			key:   indvidualJobStatusPath(jm.TrainingID, i),
			watch: individualWatch,
		}

		individualJobStatusChannels = append(individualJobStatusChannels, &item)
	}

	for _, ch := range individualJobStatusChannels {
		go func(individualCh *IndividualStatusChannelsWatch) {
			for wresp := range individualCh.watch {

				if err := wresp.Err(); err != nil {
					jm.metrics.failedETCDWatchCounter.Add(1)
					logr.WithError(err).Errorf("watch failed with error, this should not have happened for path %s", individualCh.key)
				}

				if wresp.Canceled {
					logr.Warnf("Received cancel event against the watch channel %v and path %s , so breaking out of the watch logic with possible error %v", wresp, overallJobStatusPath(jm.TrainingID), wresp.Err())
					done <- fmt.Sprintf("received cancel with error %v", wresp.Err())
				}

				if wresp.IsProgressNotify() {
					etcdLearnerProgressNotificationCounter++
					if etcdLearnerProgressNotificationCounter >= etcdProgressNotificationLogFrequency {
						logr.Infof("progress notification for watch on learner status %s", individualCh.key)
						etcdLearnerProgressNotificationCounter = 0
					}
				}

				for _, ev := range wresp.Events {
					switch ev.Type {
					case clientv3.EventTypePut:
						logr.Infof("Received update on MATCHING LEARNER on path %s, with event %v , with new value as %v", string(ev.Kv.Key), ev.Type, string(ev.Kv.Value))
						finished, error := jm.processUpdateLearnerStatus(string(ev.Kv.Key), string(ev.Kv.Value), logr)
						if error != nil {
							logr.WithError(error).Warnf("Error occurred when trying to process learner status for path %s with value %s , skipping ... ", string(ev.Kv.Key), string(ev.Kv.Value))
						}
						if error != nil && finished {
							done <- fmt.Sprintf("Received call to stop watching learner because %s has received a value of %s", ev.Kv.Key, ev.Kv.Value)
						}
					case clientv3.EventTypeDelete:
						logr.Warn("wasn't expecting to get a delete message on path %s", string(ev.Kv.Key))
					default:
						logr.Warn("wasn't expecting to get a default message on path %s", string(ev.Kv.Key))
					}
				}
			}

		}(ch)
	}

	time.AfterFunc(5*time.Second, func() {
		logr.Info("Sending a one time request to check if the paths already exist")

		//check if individual job status path exists for indvidual learners as well and if so send a message to one time watch
		for i := 1; i <= jm.NumLearners; i++ {
			individualJobStatusResponse, error := jm.etcdClient.Get(indvidualJobStatusPath(jm.TrainingID, i), logr)
			if error != nil {
				logr.WithError(error).Errorf("(monitorLearner) Learner with path %s : got error while doing a one time get", indvidualJobStatusPath(jm.TrainingID, i))
			}
			if individualJobStatusResponse != nil && len(individualJobStatusResponse) > 0 {
				logr.Infof("(monitorLearner) Learner with path %s : was already present with val %s while doing a one time get, sending signal downstream", indvidualJobStatusPath(jm.TrainingID, i), individualJobStatusResponse[0].Value)
				finished, error := jm.processUpdateLearnerStatus(individualJobStatusResponse[0].Key, individualJobStatusResponse[0].Value, logr)
				if error != nil {
					logr.WithError(error).Warnf("Error occurred when trying to process learner status first time for path %s with value %s , skipping ... ", individualJobStatusResponse[0].Key, individualJobStatusResponse[0].Value)
				}
				if error != nil && finished {
					done <- fmt.Sprintf("Received call to stop learner from  timer because %s has received a value of %s", individualJobStatusResponse[0].Key, individualJobStatusResponse[0].Value)
				}
			}
		}
	})

	for {
		select {
		case reason := <-done:
			logr.Infof("Finished processing and getting out because of reason %s", reason)
			close(done)
			return

		case wresp := <-overallJobStatusChannel:
			if err := wresp.Err(); err != nil {
				jm.metrics.failedETCDWatchCounter.Add(1)
				logr.WithError(err).Errorf("watch failed with error, this should not have happened for path %s", overallJobStatusPath(jm.TrainingID))
			}

			if wresp.Canceled {
				logr.Warnf("Recieved cancel event against the watch channel %v and path %s , so breaking out of the watch logic with possible error %v", wresp, overallJobStatusPath, wresp.Err())
				done <- fmt.Sprintf("received cancel event on path %s with error %v", overallJobStatusPath(jm.TrainingID), wresp.Err())
			}
			if wresp.IsProgressNotify() {
				etcdJobProgressNotificationCounter++
				if etcdJobProgressNotificationCounter >= etcdProgressNotificationLogFrequency {
					logr.Infof("progress notification for watch for path %s", overallJobStatusPath(jm.TrainingID))
					etcdJobProgressNotificationCounter = 0
				}
			}
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					logr.Infof("Received update on MATCHING JOB on path %s, with event %v , with new value as %v", ev.Kv.Key, ev.Type, ev.Kv.Value)
					finished := jm.processUpdateJobStatus(string(ev.Kv.Value), failedTrainerConnectivityCounter, logr)
					if finished {
						done <- fmt.Sprintf("Received call to stop watching job because %s has received a value of %s", ev.Kv.Key, ev.Kv.Value)
					}
				case clientv3.EventTypeDelete:
					logr.Warn("wasn't expecting to get a delete message on path %s", ev.Kv.Key)
				default:
					logr.Warn("wasn't expecting to get a default message on path %s", ev.Kv.Key)
				}
			}
		}
	}
}

//gets triggered when the /status node is updated
//This function updates the overall job status with trainer and calls LCM to clean up the job when necessary
//This function should only return true if the job needs no further status monitoring
func (jm *JobMonitor) processUpdateJobStatus(currStatus string, failedTrainerConnectivityCounter metrics.Counter, logr *logger.LocLoggingEntry) bool {
	logr.Infof("(processUpdateJobStatus) got triggered with the current status %s", currStatus)
	//Variable to notify whether the job needs further status monitoring
	markComplete := false
	statusUpdate := client.GetStatus(currStatus, logr)

	// prepare error/status message
	statusUpdate.StatusMessage = prepareStatusMessage(statusUpdate.ErrorCode, statusUpdate.StatusMessage)

	status := statusUpdate.Status
	error := updateJobStatusInTrainer(jm.TrainingID, jm.UserID, statusUpdate, logr)
	if error != nil {
		logr.WithError(error).Errorf("Failed to write the status %s for training %s to trainer", status, jm.TrainingID)
	}

	//if native distribution and status of the entire job is complete then kill the deployed job
	if status == grpc_trainer_v2.Status_COMPLETED || status == grpc_trainer_v2.Status_FAILED || status == grpc_trainer_v2.Status_HALTED {
		logr.Infof("(processUpdateJobStatus) overall status of the job was set up as %s and native distribution status was %s", currStatus, jm.UseNativeDistribution)
		if jm.UseNativeDistribution {
			logr.Debugf("(processUpdateJobStatus) No need to wait for all learners to terminate. Already updated status. Killing job %s", jm.TrainingID)
			err := KillDeployedJob(jm.TrainingID, jm.UserID, jm.JobName, logr)
			if err != nil {
				logr.WithError(err).Errorf("(processUpdateJobStatus) failed to kill the deployed job %s", jm.TrainingID)
			}
			markComplete = true
			return markComplete
		}
		//Job has completed, now wait 1 minute for all learners to upload logs and clean themselves up
		if atomic.LoadUint64(&jm.numTerminalLearners) < uint64(jm.NumLearners) {
			logr.Debugf("(processUpdateJobStatus) Sleeping for 60s to allow all remaining learners to complete")
			time.Sleep(60 * time.Second)
		}
		// check if they cleaned themselves up, and log it.  Teardown happens either way.
		if atomic.LoadUint64(&jm.numTerminalLearners) < uint64(jm.NumLearners) {
			logr.Debugf("(processUpdateJobStatus) Killing remaining learners in %s", jm.TrainingID)
		} else {
			logr.Debugf("(processUpdateJobStatus) All learners of %s have completed. It can now be safely killed", jm.TrainingID)
		}
		err := KillDeployedJob(jm.TrainingID, jm.UserID, jm.JobName, logr)
		if err != nil {
			logr.WithError(err).Errorf("(processUpdateJobStatus) failed to kill the deployed job %s", jm.TrainingID)
		}
		markComplete = true
	}

	return markComplete
}

//This function processes an update to learner status, i.e. it updates the overall job status
//This function should return true only if the learner needs no further status monitoring
func (jm *JobMonitor) processUpdateLearnerStatus(learnerStatusPath string, learnerStatusValue string, logr *logger.LocLoggingEntry) (bool, error) {
	statusUpdate := client.GetStatus(learnerStatusValue, logr)
	learnerStatus := statusUpdate.Status
	logr.Infof("(processUpdateLearnerStatus) got triggered with the current path %s and value %s (status %s)", learnerStatusPath, learnerStatusValue, learnerStatus)
	//Variable to notify whether the learner needs further status monitoring
	markComplete := false

	response, err := jm.etcdClient.Get(overallJobStatusPath(jm.TrainingID), logr)
	if err != nil {
		//FIXME not sure if we should be returning false or true here, since false means stop the watch
		return markComplete, err
	}
	if response == nil || len(response) == 0 {
		return markComplete, fmt.Errorf("(processUpdateLearnerStatus) while processing update from learner, the value at overall job status path %s was empty, the default value is NOT_STARTED", overallJobStatusPath(jm.TrainingID))
	}
	currentOverallJobStatus := response[0].Value
	if jm.isTransitionAllowed(currentOverallJobStatus, learnerStatus.String()) {
		logr.Infof("Transition was allowed, changing overall status of job from %s to learners status %s", currentOverallJobStatus, learnerStatus)
		jm.etcdClient.CompareAndSwap(overallJobStatusPath(jm.TrainingID), learnerStatus.String(), currentOverallJobStatus, logr)
	} else {
		logr.Warnf("Transition not allowed job from overall job status %s to learner status %s", currentOverallJobStatus, learnerStatus)
	}
	//keep an eye on idividual learners as well, if they terminate then check if all of them are done then check if job can be terminated
	if learnerStatus == grpc_trainer_v2.Status_COMPLETED || learnerStatus == grpc_trainer_v2.Status_FAILED || learnerStatus == grpc_trainer_v2.Status_HALTED {
		atomic.AddUint64(&jm.numTerminalLearners, 1)
		markComplete = true
	}
	return markComplete, err
}

func jobLearnerStatusPathPattern(trainingID string, numLearners int) string {
	pattern := fmt.Sprintf("%s/%s/%s[0-%d]/%s/", trainingID, zkLearners, zkLearner, numLearners, zkStatus)
	return pattern
}

func overallJobStatusPath(trainingID string) string {
	return trainingID + "/" + zkStatus
}

func indvidualJobStatusPath(trainingID string, learnerNum int) string {
	return fmt.Sprintf("%s/%s/%s%d/%s/", trainingID, zkLearners, zkLearner, learnerNum, zkStatus)
}

func jobBasePath(trainingID string) string {
	return trainingID + "/"
}

//Set the Job status to FAILED when the container can't find the image
func (jm *JobMonitor) monitorLearnerForFailedImagePull(logr *logger.LocLoggingEntry) {
	for {
		logr.Debugf("(monitorLearnerForFailedImagePull) Checking Learners for failed image pull")

		selector := "training_id==" + jm.TrainingID
		defaultBackoff := backoff.NewExponentialBackOff()
		defaultBackoff.MaxElapsedTime = 0 //will keep infinitely retrying
		defaultBackoff.MaxInterval = 30 * time.Second

		var pods *v1core.PodList
		var err error
		backoff.RetryNotify(func() error {
			pods, err = jm.k8sClient.Core().Pods(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
			return err
		}, defaultBackoff, func(err error, t time.Duration) { jm.metrics.failedK8sConnectivityCounter.Add(1) })

		atleastSingleContainerWaiting := false
		if len(pods.Items) > 0 {
			containerStatuses := pods.Items[0].Status.ContainerStatuses
			for _, containerStatus := range containerStatuses {
				if containerStatus.State.Waiting != nil {
					atleastSingleContainerWaiting = true
					reason := containerStatus.State.Waiting.Reason
					if reason == "ErrImagePull" {
						logr.Errorf("(monitorLearnerForFailedImagePull) Failed to start container %s: %s", containerStatus.Name, containerStatus.State.Waiting.Message)
						jm.metrics.failedImagePullK8sErrorCounter.Add(1)

						updateJobStatusOnError(jm.TrainingID, jm.UserID, client.ErrCodeImagePull, service.StatusMessages_INTERNAL_ERROR.String(), logr)
						return
					}
				}
			}

		}
		if !atleastSingleContainerWaiting {
			break
		}
	}

}

//KillDeployedJob ... Contact the LCM and kill training job
func KillDeployedJob(trainingID string, userID string, jobName string, logr *logger.LocLoggingEntry) error {
	time.Sleep(10 * time.Second)
	logr.Infof("(killDeployedJob) Sending job kill request to LCM for %s", trainingID)
	jobKillReq := &service.JobKillRequest{Name: jobName, TrainingId: trainingID, UserId: userID}
	lcm, err := lcmClient.NewLcm(nil)
	if err != nil {
		logr.Errorln("(KillDeployedJob) Cannot create lcm service client: ", err.Error())
		return err
	}
	defer lcm.Close()

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute
	defaultBackoff.MaxInterval = 5 * time.Second

	err = backoff.Retry(func() error {
		_, err = lcm.Client().KillTrainingJob(context.Background(), jobKillReq)
		if err != nil {
			logr.WithError(err).Errorf("Failed to send request to LCM to garbage collect Training Job %s. Retrying", trainingID)
		}
		return err
	}, defaultBackoff)

	if err != nil {
		logr.WithError(err).Errorf("(killDeployedJob) Successfully sent request to LCM to garbage collect Failed to send request to LCM to garbage collect Training Job %s. Already retried several times.", trainingID)
		return err
	}

	return err
}

func learnerSummaryMetricsPath(trainingID string, learnerID int) string {
	return fmt.Sprintf("%s/learners/learner_%d/%s", trainingID, learnerID, "summary_metrics")
}

func initTransitionMap() map[string]([]string) {
	transistionMap := make(map[string]([]string))
	allowDOWNLOADING := []string{grpc_trainer_v2.Status_PENDING.String(), grpc_trainer_v2.Status_NOT_STARTED.String()}
	allowPROCESSING := []string{grpc_trainer_v2.Status_DOWNLOADING.String(), grpc_trainer_v2.Status_PENDING.String()}
	allowSTORING := []string{grpc_trainer_v2.Status_PROCESSING.String(), grpc_trainer_v2.Status_DOWNLOADING.String(), grpc_trainer_v2.Status_PENDING.String(), grpc_trainer_v2.Status_NOT_STARTED.String()}
	allowCOMPLETED := []string{grpc_trainer_v2.Status_STORING.String(), grpc_trainer_v2.Status_PROCESSING.String(), grpc_trainer_v2.Status_DOWNLOADING.String(), grpc_trainer_v2.Status_PENDING.String(), grpc_trainer_v2.Status_NOT_STARTED.String()}
	allowFAILED := []string{grpc_trainer_v2.Status_STORING.String(), grpc_trainer_v2.Status_PROCESSING.String(), grpc_trainer_v2.Status_DOWNLOADING.String(), grpc_trainer_v2.Status_PENDING.String(), grpc_trainer_v2.Status_NOT_STARTED.String()}
	allowHALTED := []string{grpc_trainer_v2.Status_STORING.String(), grpc_trainer_v2.Status_PROCESSING.String(), grpc_trainer_v2.Status_DOWNLOADING.String(), grpc_trainer_v2.Status_PENDING.String(), grpc_trainer_v2.Status_NOT_STARTED.String()}

	transistionMap[grpc_trainer_v2.Status_DOWNLOADING.String()] = allowDOWNLOADING
	transistionMap[grpc_trainer_v2.Status_PROCESSING.String()] = allowPROCESSING
	transistionMap[grpc_trainer_v2.Status_STORING.String()] = allowSTORING
	transistionMap[grpc_trainer_v2.Status_COMPLETED.String()] = allowCOMPLETED
	transistionMap[grpc_trainer_v2.Status_FAILED.String()] = allowFAILED
	transistionMap[grpc_trainer_v2.Status_HALTED.String()] = allowHALTED
	return transistionMap
}

func (jm *JobMonitor) isTransitionAllowed(fromStatus string, toStatus string) bool {
	validFroms := jm.trMap[toStatus]
	for _, allowed := range validFroms {
		if fromStatus == allowed {
			return true
		}
	}
	return false
}
