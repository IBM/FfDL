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
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.ibm.com/ffdl/ffdl-core/commons/config"
	"github.ibm.com/ffdl/ffdl-core/commons/logger"
	"github.ibm.com/ffdl/ffdl-core/commons/metricsmon"
	"github.ibm.com/ffdl/ffdl-core/commons/service"
	"github.ibm.com/ffdl/ffdl-core/lcm/coord"
	jobmonitor "github.ibm.com/ffdl/ffdl-core/jobmonitor/jobmonitor"
	"github.ibm.com/ffdl/ffdl-core/lcm/lcmconfig"
	"github.ibm.com/ffdl/ffdl-core/trainer/trainer/grpc_trainer_v2"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/clientv3"
	"github.com/go-kit/kit/metrics"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
)

// Confuse `go vet' to not check this `Errorf' call. :(
// See https://github.com/grpc/grpc-go/issues/90
var gerrf = grpc.Errorf

var (
	//NativeFrameworks which support native distribution
	NativeFrameworks = []string{"tensorflow", "caffe2", "mxnet", "horovod"}
	totalTrainingCounter, finishedTrainingCounter,
	failedToLaunchTrainingsCounter metrics.Counter
)

//Service LCM manages the lifecycle of the entire distributed deep learning job
type Service interface {
	service.LifecycleManagerServer
	service.LifecycleHandler
}

type lcmService struct {
	service.Lifecycle
	k8sClient   kubernetes.Interface
	jmK8sClient kubernetes.Interface
	etcdClient  coord.Coordinator
}

//NewService is a constructor to initialize LCM
func NewService() (Service, error) {
	s, err := newService()
	return s, err
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
	lcmRestartCounter := metricsmon.NewCounter("lcm_restart_total", "Metrics for lcm restarts because of failures", []string{reason})

	// assert necessary config keys
	config.FatalOnAbsentKey(config.ETCDEndpoints)

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 1 * time.Minute

	var coordinator coord.Coordinator
	var err error
	err = backoff.RetryNotify(func() error {
		coordinator, err = coord.NewCoordinator(coord.Config{Endpoints: config.GetEtcdEndpoints(), Prefix: config.GetEtcdPrefix(),
			Cert: config.GetEtcdCertLocation(), Username: config.GetEtcdUsername(), Password: config.GetEtcdPassword()}, logr)
		return err
	}, defaultBackoff, func(err error, t time.Duration) {
		lcmRestartCounter.With(reason, "etcd").Add(1)
	})

	if err != nil {
		logr.WithError(err).Errorf("Failed to connect to etcd while creating new lcm service with cfg %v", "")
		failedToLaunchTrainingsCounter.With(reason, errCodeEtcdConnection).Add(1)
		return nil, err
	}

	var k8sClient kubernetes.Interface
	err = backoff.RetryNotify(func() error {
		k8sClient, err = kubernetes.NewForConfig(lcmconfig.GetKubernetesConfig())
		return err
	}, defaultBackoff, func(err error, t time.Duration) {
		lcmRestartCounter.With(reason, "k8s").Add(1)
	})

	if err != nil {
		logr.WithError(err).Errorf("Failed to create a kubernetes client: %v", lcmconfig.GetKubernetesConfig())
		failedToLaunchTrainingsCounter.With(reason, errCodeK8SConnection).Add(1)
		return nil, err
	}

	var jmK8sClient kubernetes.Interface

	//Recognize a hybrid deployment by looking at DLAAS_ENV which is pointed by config.EnvKey
	if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {

		//Configure LCM to recognize minikube
		jmKubesConfig, err := k8srest.InClusterConfig()
		if err != nil {
			logr.WithError(err).Errorf("Failed to read kubernetes config for Job Monitor")
			return nil, err
		}
		jmK8sClient, err = kubernetes.NewForConfig(jmKubesConfig)
		if err != nil {
			logr.WithError(err).Errorf("Failed to create a kubernetes client for Job Monitor: %v", jmKubesConfig)
			return nil, err
		}

	}

	s := &lcmService{
		k8sClient:   k8sClient,
		jmK8sClient: jmK8sClient,
		etcdClient:  coordinator,
	}

	s.RegisterService = func() {
		service.RegisterLifecycleManagerServer(s.Server, s)
	}

	return s, nil
}

//Deploys a training job in DLaaS. Retained for compatibility with other DLaaS microservices
func (s *lcmService) DeployTrainingJob(ctx context.Context, req *service.JobDeploymentRequest) (*service.JobDeploymentResponse, error) {
	logr := logger.LocLogger(s.logWithJobDeploymentRequest(req))

	totalTrainingCounter.With("framework", req.Framework).Add(1)
	err := updateJobStatus(req.TrainingId, grpc_trainer_v2.Status_PENDING, req.UserId, service.StatusMessages_NORMAL_OPERATION.String(), errCodeNormal, logr)
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
	logr := logger.LocLogger(s.logWithJobHaltRequest(req))
	logr.Infof("Halting training job: %s", req.TrainingId)
	path := req.TrainingId + "/halt"
	success, error := s.etcdClient.PutIfKeyMissing(path, "", logr)
	if error != nil {
		logr.WithError(error).Errorf("Failed to update the halt training job status on path %s for training job %s", path, req.TrainingId)
		return nil, error
	}
	if !success {
		logr.Warnf("While upadating halt for training job %s at path %s , the path already exists", req.TrainingId, path)
	}
	counter.With(progress, "etcdKeysDeleted").Add(1)

	return &service.JobHaltResponse{}, nil
}

//default deploy job function.
func (s *lcmService) deployDistributedTrainingJob(ctx context.Context, req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) {
	numLearners := int(req.GetResources().Learners)
	useNativeDistribution := false

	if numLearners < 1 {
		numLearners = 1
	}
	logr.Infof("Deploying training job: %s with %d learners", req.TrainingId, numLearners)

	//Disabling for now. To be re-enabled later after C4 and C5 permission diagnosis is complete
	/*continueDeploy := s.currentResourceSnapshot(req, numLearners, logr)

	  if !continueDeploy {
	      return
	  }*/

	// Initialize distributed training information in Zookeeper
	err := createEtcdNodes(s, req.Name, req.UserId, req.TrainingId, numLearners, req.Framework, logr)
	if err != nil {
		failedToLaunchTrainingsCounter.With(reason, errCodeEtcdConnection).Add(1)
		logr.WithError(err).Errorf("(LCM deployDistributedTrainingJob) Failed to create etcd nodes necessary to deploy a training job. Error is %s", err.Error())
		errUpd := updateJobStatus(req.TrainingId, grpc_trainer_v2.Status_FAILED, req.UserId, service.StatusMessages_INTERNAL_ERROR.String(), errCodeEtcdConnection, logr)
		if errUpd != nil {
			logr.WithError(err).Errorf("(deployDistributedTrainingJob) Before deploying job, error while calling Trainer service client update with id %s", req.TrainingId)

		}
		return //short circuit the code here, since the trainer was updated it knows the job was failed
	}

	for _, n := range NativeFrameworks {
		if req.Framework == n && numLearners > 1 {
			logr.Infof("(LCM deployDistributedTrainingJob) Using native distribution. There is no need for a parameter server for job %s ", req.TrainingId)
			useNativeDistribution = true
			break
		}
	}

	logr.Infof("(LCM deployDistributedTrainingJob) Now starting to deploy job monitor to monitor training job %s", req.TrainingId)

	err = manageDistributedJob(s, req, req.TrainingId, numLearners, req.Name, req.UserId, useNativeDistribution, logr)

	if err != nil {
		failedToLaunchTrainingsCounter.With(reason, jmLaunchFailed).Add(1)
		logr.WithError(err).Errorf("(LCM deployDistributedTrainingJob) Failed to create job monitor for training job %s", req.TrainingId)
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "job monitor", logr)
		return
	}

	if numLearners > 1 && !useNativeDistribution {
		logr.Infof("(LCM deployDistributedTrainingJob) NOT using native distribution. Deploying a parameter server for job %s ", req.TrainingId)
		err = deployParameterServer(ctx, s, req)
		if err != nil {
			failedToLaunchTrainingsCounter.With(reason, psLaunchFailed).Add(1)
			//Deploying parameter server has failed. Parameter server didn't start. So update status
			handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "parameter server deployment", logr)
			return
		}
		logr.Infof("(LCM deployDistributedTrainingJob) Finished deploying parameter server and checking whether it started")

	}

	logr.Infof("(LCM deployDistributedTrainingJob) Now starting to deploy learners for training job %s", req.TrainingId)
	err = deployLearnersAndHelpers(ctx, s, req, useNativeDistribution)
	if err != nil {
		//Deploying learner helpers has failed. So update status
		failedToLaunchTrainingsCounter.With(reason, learnerLaunchFailed).Add(1)
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "learner deployment", logr)
	}
}

//Kills a currently executing training job and cleans up its zookeeper entries
func (s *lcmService) KillTrainingJob(ctx context.Context, req *service.JobKillRequest) (*service.JobKillResponse, error) {
	counter := finishedTrainingCounter.With(outcome, killed)
	counter.With(progress, started).Add(1)
	logr := logger.LocLogger(s.logWithJobKillRequest(req))
	logr.Infof("Killing training job: %s", req.Name)

	selector := "training_id==" + req.TrainingId
	var gracePeriodSeconds int64
	deleteOpts := &metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
	}

	logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes services associated with training job %s", req.TrainingId)
	svcs, err := s.k8sClient.Core().Services(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		logr.Debugf("(LCM KillTrainingJob) Services for job with name '%s' found by querying kubernetes.", req.Name)
		for _, svc := range svcs.Items {
			logr.Debugf("(LCM KillTrainingJob) Deleting service '%s'", svc.ObjectMeta.Name)
			err := s.k8sClient.Core().Services(config.GetLearnerNamespace()).Delete(svc.ObjectMeta.Name, deleteOpts)
			if err != nil {
				logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes service '%s' failed", svc.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, servicesDeletedPhaseComplete).Add(1)

	logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes deployments associated with training job %s", req.TrainingId)
	deploys, err := s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		logr.Debugf("(LCM KillTrainingJob) Deployments for job with name '%s' found by querying kubernetes.", req.Name)
		for _, deploy := range deploys.Items {
			logr.Debugf("(LCM KillTrainingJob) Deleting deployment '%s'", deploy.ObjectMeta.Name)
			err := s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).Delete(deploy.ObjectMeta.Name, deleteOpts)
			if err != nil {
				logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes deployment '%s' failed", deploy.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, deploymentsDeletedPhaseComplete).Add(1)

	logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes replica sets associated with training job %s", req.TrainingId)
	replicaSets, err := s.k8sClient.Extensions().ReplicaSets(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for _, rs := range replicaSets.Items {
			logr.Debugf("(LCM KillTrainingJob) Deleting replica set '%s'", rs.ObjectMeta.Name)
			err := s.k8sClient.Extensions().ReplicaSets(config.GetLearnerNamespace()).Delete(rs.ObjectMeta.Name, deleteOpts)
			if err != nil {
				logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes replica set '%s' failed", rs.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, replicaSetsDeletedPhaseComplete).Add(1)

	if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {
		logr.Debugf("(LCM KillTrainingJob) LCM is running in a hybrid deployment. Checking if there are minikube deployments associated with training job %s", req.TrainingId)
		deploys, err = s.jmK8sClient.Extensions().Deployments(config.GetPodNamespace()).List(metav1.ListOptions{LabelSelector: selector})
		if err == nil {
			logr.Debugf("(LCM KillTrainingJob) Deployments for job with name '%s' found on minikube.", req.Name)
			for _, deploy := range deploys.Items {
				logr.Debugf("(LCM KillTrainingJob) Deleting deployment '%s'", deploy.ObjectMeta.Name)
				err := s.jmK8sClient.Extensions().Deployments(config.GetPodNamespace()).Delete(deploy.ObjectMeta.Name, deleteOpts)
				if err != nil {
					logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes deployment '%s' failed", deploy.ObjectMeta.Name)
				}
			}
			time.Sleep(10 * time.Second)
		}

		logr.Debugf("(LCM KillTrainingJob) LCM is running in a hybrid deployment. Checking if there are minikube replica sets associated with training job %s", req.TrainingId)
		replicaSets, err := s.jmK8sClient.Extensions().ReplicaSets(config.GetPodNamespace()).List(metav1.ListOptions{LabelSelector: selector})
		if err == nil {
			for _, rs := range replicaSets.Items {
				logr.Debugf("(LCM KillTrainingJob) Deleting replica set '%s'", rs.ObjectMeta.Name)
				err := s.jmK8sClient.Extensions().ReplicaSets(config.GetPodNamespace()).Delete(rs.ObjectMeta.Name, deleteOpts)
				if err != nil {
					logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes replica set '%s' failed", rs.ObjectMeta.Name)
				}
			}
		}
	}

	// delete pods
	if !strings.Contains(config.GetDebugLearnerOptions(), config.NoCleanup) {
		logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes learner PODS associated with training job %s", req.TrainingId)
		pods, err := s.k8sClient.Core().Pods(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
		if err == nil {
			for _, pod := range pods.Items {
				logr.Debugf("(LCM KillTrainingJob) Deleting pod '%s'", pod.ObjectMeta.Name)
				err := s.k8sClient.Core().Pods(config.GetLearnerNamespace()).Delete(pod.ObjectMeta.Name, deleteOpts)
				if err != nil {
					logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes pod '%s' failed", pod.ObjectMeta.Name)
				}
			}
		}
		if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {
			pods, err := s.jmK8sClient.Core().Pods(config.GetPodNamespace()).List(metav1.ListOptions{LabelSelector: selector})
			if err == nil {
				for _, pod := range pods.Items {
					logr.Debugf("(LCM KillTrainingJob) Deleting pod '%s'", pod.ObjectMeta.Name)
					err := s.jmK8sClient.Core().Pods(config.GetPodNamespace()).Delete(pod.ObjectMeta.Name, deleteOpts)
					if err != nil {
						logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes pod '%s' failed", pod.ObjectMeta.Name)
					}
				}
			}
		}
	}
	counter.With(progress, podsDeletedPhaseComplete).Add(1)

	logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes learner persistent volume claims associated with training job %s", req.TrainingId)
	claims, err := s.k8sClient.Core().PersistentVolumeClaims(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for _, claim := range claims.Items {
			logr.Debugf("(LCM KillTrainingJob) Deleting persistent volume claim '%s'", claim.ObjectMeta.Name)
			err := s.k8sClient.Core().PersistentVolumeClaims(config.GetLearnerNamespace()).Delete(claim.ObjectMeta.Name, deleteOpts)
			if err != nil {
				logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes persistent volume '%s' failed", claim.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, pvsDeletedPhaseComplete).Add(1)

	logr.Debugf("(LCM KillTrainingJob) Checking if there are kubernetes learner COS mount secrets associated with training job %s", req.TrainingId)
	secrets, err := s.k8sClient.Core().Secrets(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})
	if err == nil {
		for _, secret := range secrets.Items {
			logr.Debugf("(LCM KillTrainingJob) Deleting Secret '%s'", secret.ObjectMeta.Name)
			err := s.k8sClient.Core().Secrets(config.GetLearnerNamespace()).Delete(secret.ObjectMeta.Name, deleteOpts)
			if err != nil {
				logr.WithError(err).Errorf("(LCM KillTrainingJob) Deleting kubernetes Secret '%s' failed", secret.ObjectMeta.Name)
			}
		}
	}
	counter.With(progress, secretsDeletedPhaseComplete).Add(1)

	//After Deleting the application, delete the etcd directory.
	s.etcdClient.DeleteKeyWithOpts(req.TrainingId, logr, clientv3.WithPrefix())
	counter.With(progress, etcdKeysDeletedPhaseComplete).Add(1)

	go s.resourceSnapshotOnDeletion(req, logr)

	return &service.JobKillResponse{}, nil
}

//Wrapper function for LCM's KillTrainingJob
func (s *lcmService) killDeployedJob(jobName string, trainingID string, userID string) error {
	job := &service.JobKillRequest{Name: string(jobName), TrainingId: trainingID, UserId: userID}
	_, err := s.KillTrainingJob(context.Background(), job)
	return err
}

//manages a DLaaS training job
func manageDistributedJob(s *lcmService, req *service.JobDeploymentRequest, trainingID string, numLearners int, jobName string, userID string, useNativeDistribution bool, logr *logger.LocLoggingEntry) error {

	envVars, jmLabels := populateJobMonitorEnvVariablesAndLabels(req, trainingID, jobName, userID, numLearners, useNativeDistribution)
	deploySpec := defineJobMonitorDeployment(req, envVars, jmLabels, logr)

	var jmDeploy *v1beta1.Deployment
	var err error

	//decide where to deploy job monitor by looking at DLAAS_ENV, which is pointed to by config.EnvKey
	switch viper.GetString(config.LCMDeploymentKey) {
	case config.HybridEnv:
		{
			logr.Debugf("(LCM manageDistributedJob) LCM is running in a hybrid deployment. Deploying Job Monitor in minikube instead of the learner namespace")
			_, errJMDeploy := s.jmK8sClient.Extensions().Deployments(config.GetPodNamespace()).Create(deploySpec)
			if errJMDeploy != nil {
				logr.WithError(errJMDeploy).Errorf("(LCM manageDistributedJob) Could not connect to minikube and/or deploy job monitor for Training Job %s. Problem may either be in reaching the kubernetes API server or in creating a deployment", req.TrainingId)
				return errJMDeploy
			}
		}
	case config.LocalEnv:
		{
			logr.Debugf("(LCM manageDistributedJob) LCM is running in a local deployment. Deploying Job Monitor as a thread spawned by the LCM")

			statsdClient := metricsmon.NewStatsdClient("mock_local_jobmonitor")
			jmlogr := logger.LocLogger(jobmonitor.InitLogger(trainingID, userID))
			jm, err := jobmonitor.NewJobMonitor(trainingID, userID, numLearners, jobName, useNativeDistribution, statsdClient, jmlogr)
			if err != nil {
				logr.WithError(err).Errorf("Failed to bring up job monitor for training %s", trainingID)
			}
			go jm.ManageDistributedJob(jmlogr)
			return err
		}
	default:
		{
			logr.Debugf("(LCM manageDistributedJob) LCM is running in a fully remote mode. Deploying Job Monitor in the learner namespace.")
			_, errJMDeploy := s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).Create(deploySpec)
			if errJMDeploy != nil {
				logr.WithError(errJMDeploy).Errorf("(LCM manageDistributedJob) Could not connect to learner kube cluster and/or deploy job monitor for Training Job %s. Problem may either be in reaching the kubernetes API server or in creating a deployment", req.TrainingId)
				return errJMDeploy
			}
		}
	}

	defaultBackoff := backoff.NewExponentialBackOff()
	defaultBackoff.MaxElapsedTime = 10 * time.Minute
	defaultBackoff.MaxInterval = 3 * time.Minute

	jmName := constructJMName(req.Name)
	err = backoff.Retry(func() error { // retry get with backoff. This is not an infinite loop
		logr.Debugf("(LCM manageDistributedJob) Checking whether Job Monitor Started for Training Job %s", req.TrainingId)

		if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {
			jmDeploy, err = s.jmK8sClient.Extensions().Deployments(config.GetPodNamespace()).Get(jmName, metav1.GetOptions{})
			if err != nil {
				logr.WithError(err).Errorf("(LCM manageDistributedJob) Could not query minikube and/or find Job Monitor for Training Job %s", req.TrainingId)
				return err
			}
		} else {
			jmDeploy, err = s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).Get(jmName, metav1.GetOptions{})
			if err != nil {
				logr.WithError(err).Errorf("(LCM manageDistributedJob) Could not query Kubernetes and/or find Job Monitor for Training Job %s", req.TrainingId)
				logr.Infof("WARNING : Status updates for Training Job %s will likely be incorrect.", req.TrainingId)
				return err
			}
		}

		if jmDeploy != nil {
			if jmDeploy.Status.AvailableReplicas >= 1 {
				logr.Infof("Job Monitor started for Training Job %s", req.TrainingId)
				logr.Debugf("(LCM manageDistributedJob) Job Monitor Deployment is %s\n", jmDeploy)
				return nil
			}
		} else {
			err := errors.New("For some reason, job monitor deployment is nil and no error was received while querying kubernetes for the deployment")
			logr.WithError(err).Errorf("(LCM manageDistributedJob) Internal DLaaS/Kubernetes Error")
			return err
		}
		return nil
	}, defaultBackoff)

	if err != nil {
		logr.WithError(err).Errorf("(LCM manageDistributedJob) LCM could not confirm that job monitor actually started despite trying several times. Giving up")
	}

	return err
}

func (s *lcmService) GetMetrics(ctx context.Context, req *service.GetMetricsRequest) (*service.GetMetricsResponse, error) {
	return nil, fmt.Errorf("This call is deprecated and should not be used")
}
