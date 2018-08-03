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
	"time"

	"github.com/IBM/FfDL/lcm/coord"

	"google.golang.org/grpc"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/metricsmon"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/lcm/lcmconfig"
	"github.com/IBM/FfDL/trainer/client"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/clientv3"
	"github.com/go-kit/kit/metrics"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Confuse `go vet' to not check this `Errorf' call. :(
// See https://github.com/grpc/grpc-go/issues/90
var gerrf = grpc.Errorf

var (
	//NativeFrameworks which support native distribution
	NativeFrameworks = []string{"tensorflow", "caffe2", "mxnet", "horovod"}
	totalTrainingCounter, finishedTrainingCounter,
	failedToLaunchTrainingsCounter, k8sFailureCounter metrics.Counter
)

//Service LCM manages the lifecycle of the entire distributed deep learning job
type Service interface {
	service.LifecycleManagerServer
	service.LifecycleHandler
	StopLCM()
}

type lcmService struct {
	service.Lifecycle
	k8sClient  kubernetes.Interface
	etcdClient coord.Coordinator
}

//NewService is a constructor to initialize LCM
func NewService() (Service, error) {
	s, err := newService()
	return s, err
}

func (s *lcmService) StopLCM() {
	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))
	logr.Debugf(" ###### shutting down lcm ###### ")
	s.etcdClient.Close(logr)
	s.Stop() // stop Service
}

func newService() (*lcmService, error) {
	logr := logger.LocLogger(logger.LogServiceBasic(logger.LogkeyLcmService))
	logr.Infoln("Instantiating LCM service...")
	logr.Infof("LCM Deployment Mode is %s , pod namespace %s , learner namespace %s , databroker tag %s, learner tag %s",
		viper.GetString(config.LCMDeploymentKey), config.GetPodNamespace(), config.GetLearnerNamespace(),
		viper.GetString(config.DataBrokerTagKey), viper.GetString(config.LearnerTagKey))

	totalTrainingCounter = metricsmon.NewCounter("lcm_trainings_total", "Metrics for total lcm trainings", []string{framework})
	finishedTrainingCounter = metricsmon.NewCounter("lcm_trainings_killed", "Metrics for lcm trainings that were killed ", []string{outcome, progress})
	failedToLaunchTrainingsCounter = metricsmon.NewCounter("lcm_trainings_launch_failed", "Metrics for lcm trainings that failed to launch", []string{reason})
	k8sFailureCounter = metricsmon.NewCounter("k8s_deploy_failures", "metrics for tracking k8s failures when starting trainings", []string{component})
	lcmRestartCounter := metricsmon.NewCounter("lcm_restart_total", "Metrics for lcm restarts because of failures", []string{reason})

	// assert necessary config keys
	config.FatalOnAbsentKey(config.ETCDEndpoints)

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute

	k8sClient, err := kubernetes.NewForConfig(lcmconfig.GetKubernetesConfig())

	if err != nil {
		logr.WithError(err).Errorf("Failed to create a kubernetes client: %v", lcmconfig.GetKubernetesConfig())
		lcmRestartCounter.With(reason, "k8s").Add(1)
		return nil, err
	}

	client, connectivityErr := coordinator(logr)
	if connectivityErr != nil {
		logr.WithError(connectivityErr).Errorln("failed to connect to etcd when starting, this should trigger restart of lcm")
		lcmRestartCounter.With(reason, "etcd").Add(1)
		return nil, connectivityErr
	}

	s := &lcmService{
		k8sClient:  k8sClient,
		etcdClient: client,
	}

	s.RegisterService = func() {
		service.RegisterLifecycleManagerServer(s.Server, s)
	}

	return s, nil
}

//Deploys a training job in DLaaS. Retained for compatibility with other DLaaS microservices
func (s *lcmService) DeployTrainingJob(ctx context.Context, req *service.JobDeploymentRequest) (*service.JobDeploymentResponse, error) {
	//extend the logger with required fields and this logr will be passed around
	logr := logger.LocLogger(InitLogger(req.TrainingId, req.UserId).WithFields(logrus.Fields{
		"name":      req.Name,
		"framework": req.Framework,
		"gpus":      req.Resources.Gpus,
		"cpus":      req.Resources.Cpus,
		"memory":    req.Resources.Memory,
	}))

	totalTrainingCounter.With("framework", req.Framework).Add(1)
	err := updateJobStatus(req.TrainingId, grpc_trainer_v2.Status_PENDING, req.UserId, service.StatusMessages_NORMAL_OPERATION.String(), client.ErrCodeNormal, logr)
	if err != nil {
		logr.WithError(err).Errorf("(deployDistributedTrainingJob) Before deploying job, error while calling Trainer service client update for trainingID %s , but still carrying on ", req.TrainingId)
	}

	go s.deployDistributedTrainingJob(ctx, req, logr)
	return &service.JobDeploymentResponse{Name: req.Name}, nil
}

//Stops a currently executing training job
func (s *lcmService) HaltTrainingJob(ctx context.Context, req *service.JobHaltRequest) (*service.JobHaltResponse, error) {

	counter := finishedTrainingCounter.With(outcome, halted)
	counter.With(progress, started).Add(1)
	logr := logger.LocLogger(InitLogger(req.TrainingId, req.UserId))
	logr.Infof("Halting training job: %s", req.TrainingId)

	path := req.TrainingId + "/halt"
	success, error := s.etcdClient.PutIfKeyMissing(path, "", logr)
	if error != nil {
		logr.WithError(error).Errorf("Failed to update the halt training job status on path %s for training job %s", path, req.TrainingId)
		return nil, error
	}
	if !success {
		logr.Warnf("While updating halt for training job %s at path %s , the path already exists", req.TrainingId, path)
	}
	counter.With(progress, "etcdKeysDeleted").Add(1)

	return &service.JobHaltResponse{}, nil
}

//default deploy job function.
func (s *lcmService) deployDistributedTrainingJob(ctx context.Context, req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) {

	numLearners := int(req.GetResources().Learners)
	useNativeDistribution := false //always use native since we don't support PS anymore

	if numLearners < 1 {
		numLearners = 1
	}

	logr.WithField("learners", numLearners).Infof("starting deployment of training job in lcm")

	// Initialize distributed training information in Zookeeper
	if err := createEtcdNodes(s, req.Name, req.UserId, req.TrainingId, numLearners, req.Framework, logr); err != nil {
		failedToLaunchTrainingsCounter.With(reason, client.ErrCodeEtcdConnection).Add(1)
		logr.WithError(err).Errorf("Failed to create etcd nodes necessary to deploy a training job")
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "etcd nodes creation", logr)
		return //short circuit the code here, since the trainer was updated it knows the job was failed
	}

	logr.Infof("now starting to deploy job monitor to monitor training job")
	if err := deployJobMonitor(s, req, req.TrainingId, numLearners, req.Name, req.UserId, useNativeDistribution, logr); err != nil {
		failedToLaunchTrainingsCounter.With(reason, jmLaunchFailed).Add(1)
		logr.WithError(err).Errorf("Failed to create job monitor for training job")
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "job monitor", logr)
		return
	}

	logr.Infof("now starting to deploy learners for training job")
	if err := NewTraining(ctx, s.k8sClient, req, logr).Start(); err != nil {
		//Deploying learner helpers has failed. So update status
		failedToLaunchTrainingsCounter.With(reason, learnerLaunchFailed).Add(1)
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "learner deployment", logr)
		return
	}
}

//Kills a currently executing training job and cleans up its zookeeper entries
func (s *lcmService) KillTrainingJob(ctx context.Context, req *service.JobKillRequest) (*service.JobKillResponse, error) {
	counter := finishedTrainingCounter.With(outcome, killed)
	counter.With(progress, started).Add(1)
	logr := logger.LocLogger(InitLogger(req.TrainingId, req.UserId))

	logr.Infof("Killing training job: %s", req.Name)

	selector := "training_id==" + req.TrainingId
	backgroundPropagation := metav1.DeletePropagationBackground
	backgroundDeleteOpts := &metav1.DeleteOptions{
		PropagationPolicy: &backgroundPropagation,
	}

	logr.Debugf(" Checking if there are kubernetes services associated with training job %s", req.TrainingId)
	svcs, err := s.k8sClient.CoreV1().Services(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		logr.Debugf(" Services for job with name '%s' found by querying kubernetes.", req.Name)
		for _, svc := range svcs.Items {
			logr.Infof(" Deleting service '%s'", svc.ObjectMeta.Name)
			err := s.k8sClient.CoreV1().Services(config.GetLearnerNamespace()).Delete(svc.ObjectMeta.Name, backgroundDeleteOpts)
			if err != nil {
				logr.WithError(err).Errorf(" Deleting kubernetes service '%s' failed", svc.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, servicesDeletedPhaseComplete).Add(1)

	logr.Debugf(" Checking if there are kubernetes statefulsets associated with training job %s", req.TrainingId)
	sets, err := s.k8sClient.AppsV1beta1().StatefulSets(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		logr.Debugf(" Stateful for job with name '%s' found by querying kubernetes.", req.Name)
		for _, set := range sets.Items {
			logr.Infof(" Deleting stateful '%s'", set.ObjectMeta.Name)
			err := s.k8sClient.AppsV1beta1().StatefulSets(config.GetLearnerNamespace()).Delete(set.ObjectMeta.Name, backgroundDeleteOpts)
			if err != nil {
				logr.WithError(err).Errorf(" Deleting kubernetes stateful '%s' failed", set.ObjectMeta.Name)
			}
		}
	}

	logr.Debugf(" Checking if there are kubernetes learner persistent volume claims associated with training job %s", req.TrainingId)
	claims, err := s.k8sClient.CoreV1().PersistentVolumeClaims(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for _, claim := range claims.Items {
			logr.Infof(" Deleting persistent volume claim '%s'", claim.ObjectMeta.Name)
			err := s.k8sClient.CoreV1().PersistentVolumeClaims(config.GetLearnerNamespace()).Delete(claim.ObjectMeta.Name, backgroundDeleteOpts)
			if err != nil {
				logr.WithError(err).Errorf(" Deleting kubernetes persistent volume '%s' failed", claim.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, pvsDeletedPhaseComplete).Add(1)

	logr.Debugf(" Checking if there are kubernetes learner COS mount secrets associated with training job %s", req.TrainingId)
	secrets, err := s.k8sClient.CoreV1().Secrets(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for _, secret := range secrets.Items {
			logr.Infof(" Deleting Secret '%s'", secret.ObjectMeta.Name)
			err := s.k8sClient.CoreV1().Secrets(config.GetLearnerNamespace()).Delete(secret.ObjectMeta.Name, backgroundDeleteOpts)
			if err != nil {
				logr.WithError(err).Errorf(" Deleting kubernetes Secret '%s' failed", secret.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, secretsDeletedPhaseComplete).Add(1)

	logr.Debugf(" Checking if there are kubernetes deployments associated with training job %s", req.TrainingId)
	deploys, err := s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		logr.Debugf(" Deployments for job with name '%s' found by querying kubernetes.", req.Name)
		for _, deploy := range deploys.Items {
			logr.Infof(" Deleting deployment '%s'", deploy.ObjectMeta.Name)
			err := s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).Delete(deploy.ObjectMeta.Name, backgroundDeleteOpts)
			if err != nil {
				logr.WithError(err).Errorf(" Deleting kubernetes deployment '%s' failed", deploy.ObjectMeta.Name)
			}
		}
	}

	counter.With(progress, deploymentsDeletedPhaseComplete).Add(1)

	//After Deleting the application, delete the etcd directory.
	s.etcdClient.DeleteKeyWithOpts(req.TrainingId, logr, clientv3.WithPrefix())
	counter.With(progress, etcdKeysDeletedPhaseComplete).Add(1)
	return &service.JobKillResponse{}, nil
}

//Wrapper function for LCM's KillTrainingJob
func (s *lcmService) killDeployedJob(jobName string, trainingID string, userID string) error {
	job := &service.JobKillRequest{Name: string(jobName), TrainingId: trainingID, UserId: userID}
	_, err := s.KillTrainingJob(context.Background(), job)
	return err
}

//manages a DLaaS training job
func deployJobMonitor(s *lcmService, req *service.JobDeploymentRequest, trainingID string, numLearners int, jobName string, userID string, useNativeDistribution bool, logr *logger.LocLoggingEntry) error {

	envVars, jmLabels := populateJobMonitorEnvVariablesAndLabels(req, trainingID, jobName, userID, numLearners, useNativeDistribution)
	deploySpec := defineJobMonitorDeployment(req, envVars, jmLabels, logr)

	return backoff.RetryNotify(func() error {
		_, err := s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).Create(deploySpec)
		if k8serrors.IsAlreadyExists(err) {
			logr.WithError(err).Warnf("deployment %s already exists", deploySpec.ObjectMeta.Name)
			return nil
		}
		return err
	}, k8sInteractionBackoff(), func(err error, window time.Duration) {
		logr.WithError(err).Errorf("Could not connect to learner kube cluster and/or deploy job monitor for Training Job %s. Problem may either be in reaching the kubernetes API server or in creating a deployment", req.TrainingId)
		k8sFailureCounter.With(component, "jobmonitor").Add(1)
	})
}
