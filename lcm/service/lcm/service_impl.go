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
	"time"

	"google.golang.org/grpc"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/metricsmon"
	"github.com/IBM/FfDL/commons/service"
	jobmonitor "github.com/IBM/FfDL/jobmonitor/jobmonitor"
	"github.com/IBM/FfDL/lcm/coord"
	"github.com/IBM/FfDL/lcm/lcmconfig"
	"github.com/IBM/FfDL/trainer/client"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"

	"github.com/cenkalti/backoff"
	"github.com/coreos/etcd/clientv3"
	"github.com/go-kit/kit/metrics"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	v1beta1 "k8s.io/api/apps/v1beta1"
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
		failedToLaunchTrainingsCounter.With(reason, client.ErrCodeEtcdConnection).Add(1)
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
		failedToLaunchTrainingsCounter.With(reason, client.ErrCodeK8SConnection).Add(1)
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
	//extend the logger with required fields and this logr will be passed around
	logr := logger.LocLogger(InitLogger(req.TrainingId, req.UserId).WithFields(logrus.Fields{
		"name":      req.Name,
		"framework": req.Framework,
		"gpus":      req.Resources.Gpus,
		"cpus":      req.Resources.Gpus,
		"memory":    req.Resources.Gpus,
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
	if err := manageDistributedJob(s, req, req.TrainingId, numLearners, req.Name, req.UserId, useNativeDistribution, logr); err != nil {
		failedToLaunchTrainingsCounter.With(reason, jmLaunchFailed).Add(1)
		logr.WithError(err).Errorf("Failed to create job monitor for training job")
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "job monitor", logr)
		return
	}

	logr.Infof("now starting to deploy learners for training job")
	if err := NewTraining(ctx, s.k8sClient, req, logr).Start(); err != nil {
		//Deploying learner helpers has failed. So update status
		logr.WithError(err).Debugf("Deploying learner helpers has failed. Update status.")
		failedToLaunchTrainingsCounter.With(reason, learnerLaunchFailed).Add(1)
		handleDeploymentFailure(s, req.Name, req.TrainingId, req.UserId, "learner deployment", logr)
		return
	}
	logr.Debugf("Learner deployment seemed to launch ok")
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
	counter.With(progress, deploymentsDeletedPhaseComplete).Add(1)

	if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {
		logr.Debugf(" LCM is running in a hybrid deployment. Checking if there are minikube deployments associated with training job %s", req.TrainingId)
		deploys, err = s.jmK8sClient.AppsV1beta1().Deployments(config.GetPodNamespace()).List(metav1.ListOptions{LabelSelector: selector})
		if err == nil {
			logr.Infof(" Deployments for job with name '%s' found on minikube.", req.Name)
			for _, deploy := range deploys.Items {
				logr.Debugf(" Deleting deployment '%s'", deploy.ObjectMeta.Name)
				err := s.jmK8sClient.AppsV1beta1().Deployments(config.GetPodNamespace()).Delete(deploy.ObjectMeta.Name, backgroundDeleteOpts)
				if err != nil {
					logr.WithError(err).Errorf(" Deleting kubernetes deployment '%s' failed", deploy.ObjectMeta.Name)
				}
			}
			time.Sleep(10 * time.Second)
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
			_, errJMDeploy := s.jmK8sClient.AppsV1beta1().Deployments(config.GetPodNamespace()).Create(deploySpec)
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
			logr.Infof("(LCM manageDistributedJob) LCM is running in a fully remote mode. Deploying Job Monitor in the learner namespace.")
			_, errJMDeploy := s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).Create(deploySpec)
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
			jmDeploy, err = s.jmK8sClient.AppsV1beta1().Deployments(config.GetPodNamespace()).Get(jmName, metav1.GetOptions{})
			if err != nil {
				logr.WithError(err).Errorf("(LCM manageDistributedJob) Could not query minikube and/or find Job Monitor for Training Job %s", req.TrainingId)
				return err
			}
		} else {
			jmDeploy, err = s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).Get(jmName, metav1.GetOptions{})
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
