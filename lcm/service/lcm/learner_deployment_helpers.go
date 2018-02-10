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
	"io/ioutil"
	"strconv"
	"time"

	"github.com/IBM/FfDL/lcm/service/lcm/certs"

	"github.com/IBM/FfDL/commons/config"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/commons/util"

	"github.com/spf13/viper"

	"golang.org/x/net/context"

	"k8s.io/apimachinery/pkg/util/intstr"
	//"k8s.io/client-go/pkg/api/unversioned"
	v1core "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const cosMountDriverName = "ibm/ibmc-s3fs"
const cosMountType = "mount_cos"

func defineLearnerService(name string, trainingID string) *v1core.Service {

	// Define service spec.
	serviceSpec := &v1core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"training_id": trainingID,
			},
		},
		Spec: v1core.ServiceSpec{
			Type:     v1core.ServiceTypeClusterIP,
			Selector: map[string]string{"app": name},
			Ports: []v1core.ServicePort{
				v1core.ServicePort{
					Name:     "grpc",
					Protocol: v1core.ProtocolTCP,
					Port:     workerPort,
				},
				v1core.ServicePort{
					Name:     "ssh",
					Protocol: v1core.ProtocolTCP,
					Port:     sshPort,
				},
			},
		},
	}

	// Is this needed for a kubernetes service object?
	setServiceTypeLabel(&serviceSpec.ObjectMeta, "dlaas-learner")

	return serviceSpec
}

func populateETCDEnvVariables(trainingID string) []v1core.EnvVar {
	getEnvVarFromLCMSecret := func(lookupkey string) v1core.EnvVar {
		return v1core.EnvVar{
			Name: lookupkey,
			ValueFrom: &v1core.EnvVarSource{
				SecretKeyRef: &v1core.SecretKeySelector{
					Key: lookupkey,
					LocalObjectReference: v1core.LocalObjectReference{
						Name: "lcm-secrets",
					},
				},
			},
		}
	}

	learnerZkDir := learnerEtcdBasePath(trainingID)

	etcdEnvVars := []v1core.EnvVar{
		getEnvVarFromLCMSecret("DLAAS_ETCD_ADDRESS"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_USERNAME"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PASSWORD"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PREFIX"),

		v1core.EnvVar{Name: "ZK_DIR", Value: learnerZkDir},
		v1core.EnvVar{Name: "ZK_LOCK_PATH", Value: learnerZkDir + "/lock"},
		v1core.EnvVar{Name: "ZK_COUNTER_PATH", Value: learnerZkDir + "/counter"},
		v1core.EnvVar{Name: "ZNODE_NAME", Value: "learnershard"},
	}

	return etcdEnvVars
}

//Populate all the environment variables used to deploy learner jobs on Kubernetes
func populateLearnerEnvVariablesAndLabels(req *service.JobDeploymentRequest, PSServicName string, serviceNames []string, numLearners int,
	learnerID int, useNativeDistribution bool, logr *logger.LocLoggingEntry) []v1core.EnvVar {

	envVars := make([]v1core.EnvVar, 0, len(req.EnvVars))
	for k, v := range req.EnvVars {
		envVars = append(envVars, v1core.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	lcmEnvVars := []v1core.EnvVar{
		v1core.EnvVar{Name: "TRAINING_ID", Value: req.TrainingId},
		v1core.EnvVar{Name: "DLAAS_JOB_ID", Value: req.TrainingId},
		v1core.EnvVar{Name: "DLAAS_PLATFORM", Value: "kubernetes"},
	}

	distributedEnvVars := make([]v1core.EnvVar, 0)
	if numLearners > 1 {
		distributedEnvVars = []v1core.EnvVar{
			v1core.EnvVar{Name: "PARAMSERVER_HOST", Value: PSServicName},
			v1core.EnvVar{Name: "PARAMSERVER_PORT", Value: strconv.Itoa(int(psPort))},
			v1core.EnvVar{Name: "PARAMSERVER_JOBID", Value: "1111"},
			v1core.EnvVar{Name: "GLOBAL_CURSOR_ZNODE", Value: config.GetEtcdPrefix() + req.TrainingId + "/" + zkGlobalCursor + "/" + zkGCState},
			v1core.EnvVar{Name: "NUM_LEARNERS", Value: strconv.Itoa(numLearners)},
		}
	}

	if useNativeDistribution {
		isDistTF := v1core.EnvVar{Name: "IS_DISTRIBUTED_TF", Value: "1"}
		envVars = append(envVars, isDistTF)
	}

	serviceNamesVars := make([]v1core.EnvVar, 0)
	for i, v := range serviceNames {
		serviceNamesVars = append(serviceNamesVars, v1core.EnvVar{
			Name:  "SERVICE_NAME_" + strconv.Itoa(i+1),
			Value: v,
		})
	}

	envVars = append(envVars, lcmEnvVars...)
	envVars = append(envVars, distributedEnvVars...)
	envVars = append(envVars, serviceNamesVars...)
	envVars = append(envVars, v1core.EnvVar{Name: "LEARNER_ID", Value: strconv.Itoa(learnerID)})

	return envVars

}

func defineLearnerAndHelperObjects(s *lcmService, req *service.JobDeploymentRequest, useNativeDistribution bool, logr *logger.LocLoggingEntry) ([]*v1core.Service, []*v1core.PersistentVolumeClaim, []*v1beta1.Deployment, []*v1core.Secret, error) {
	serviceSpecs := []*v1core.Service{}
	volumeSpecs := []*v1core.Volume{}
	deploymentSpecs := []*v1beta1.Deployment{}
	secretSpecs := []*v1core.Secret{}

	volumeClaimSpecs := []*v1core.PersistentVolumeClaim{}

	numLearners := int(req.GetResources().Learners)
	if numLearners < 1 {
		logr.Debugf("(LCM deployLearners) A numLearners value less than 1 was received.")
		numLearners = 1
	}

	// Kubernetes service per learner, only in the case of distributed training.
	isDistributedTraining := numLearners > 1
	if isDistributedTraining {
		for learnerID := 1; learnerID <= numLearners; learnerID++ {
			name := constructLearnerServiceName(learnerID, req.Name)
			spec := defineLearnerService(name, req.TrainingId)
			serviceSpecs = append(serviceSpecs, spec)
		}
	}

	// Determine if a dynamic external volume should be used.
	storageSize := getStorageSize(req.Resources)
	logr.Debugf("Requested storage for job of size %d bytes", storageSize)
	useDynamicExternalVolume := storageSize > 0

	// Determine if a static external volume should be used.
	staticVolume := getStaticVolume(logr)
	logr.Debugf("Static volume for job: %s", staticVolume)
	useStaticExternalVolume := len(staticVolume) > 0

	// Determine whether to split learner.
	useSplitLearner := useDynamicExternalVolume || useStaticExternalVolume

	// Define volumes
	for learnerID := 1; learnerID <= numLearners; learnerID++ {
		volume := v1core.Volume{
			Name:         "jobdata",
			VolumeSource: v1core.VolumeSource{}, // to be filled in below
		}
		if useStaticExternalVolume {
			logr.Debugf("(LCM deployLearners) using static external volume %s", staticVolume)
			volume.VolumeSource = v1core.VolumeSource{
				PersistentVolumeClaim: &v1core.PersistentVolumeClaimVolumeSource{ClaimName: staticVolume},
			}
		} else if useDynamicExternalVolume {
			logr.Debugf("(LCM deployLearners) using dynamic external volume.")
			labels := map[string]string{"training_id": req.TrainingId}
			claim := constructVolumeClaim(constructLearnerVolumeClaimName(learnerID, req.Name), config.GetLearnerNamespace(), storageSize, labels)
			volumeClaimSpecs = append(volumeClaimSpecs, claim)
			volume.VolumeSource = v1core.VolumeSource{
				PersistentVolumeClaim: &v1core.PersistentVolumeClaimVolumeSource{ClaimName: claim.Name},
			}
		} else {
			logr.Debugf("(LCM deployLearners) using pod emptydir volume.")
			volume.VolumeSource = v1core.VolumeSource{
				EmptyDir: &v1core.EmptyDirVolumeSource{},
			}
		}
		volumeSpecs = append(volumeSpecs, &volume)
	}

	// prepare Cloud Object Storage mount(s)
	mountTrainingDataStoreInLearner := req.EnvVars["DATA_STORE_TYPE"] == cosMountType
	trainingMountSecretName := "cossecretdata-" + req.Name
	if mountTrainingDataStoreInLearner {
		// create secret
		spec := &v1core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "extensions/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      trainingMountSecretName,
				Namespace: config.GetLearnerNamespace(),
				Labels:    map[string]string{"training_id": req.TrainingId},
			},
			Type: cosMountDriverName,
			StringData: map[string]string{
				"access-key": req.EnvVars["DATA_STORE_USERNAME"],
				"secret-key": req.EnvVars["DATA_STORE_APIKEY"],
			},
		}
		secretSpecs = append(secretSpecs, spec)
	}
	mountResultsStoreInLearner := req.EnvVars["RESULT_STORE_TYPE"] == cosMountType
	resultsMountSecretName := "cossecretresults-" + req.Name
	if mountResultsStoreInLearner {
		// create secret
		spec := &v1core.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "extensions/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      resultsMountSecretName,
				Namespace: config.GetLearnerNamespace(),
				Labels:    map[string]string{"training_id": req.TrainingId},
			},
			Type: cosMountDriverName,
			StringData: map[string]string{
				"access-key": req.EnvVars["RESULT_STORE_USERNAME"],
				"secret-key": req.EnvVars["RESULT_STORE_APIKEY"],
			},
		}
		secretSpecs = append(secretSpecs, spec)
	}
	sshSecretName := fmt.Sprintf("jobsshcert-%s", req.Name)
	sshSecret, err := certs.GenerateSSHCertAsK8sSecret(sshSecretName, req.TrainingId, req.Framework, req.Version)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	mountSSHCertsAsSecrets := false
	if sshSecret != nil && err == nil {
		mountSSHCertsAsSecrets = true
		secretSpecs = append(secretSpecs, sshSecret)
	}

	// Define helper and learner deployments
	for learnerID := 1; learnerID <= numLearners; learnerID++ {
		logr.Debugf("constructing spec for learner %d/%d", learnerID, numLearners)

		learnerNodeBasePath := learnerNodeEtcdBasePath(req.TrainingId, learnerID)
		learnerNodeStatusPath := learnerNodeEtcdStatusPath(req.TrainingId, learnerID)
		summaryMetricsPath := learnerSummaryMetricsPath(req.TrainingId, learnerID)

		// Volume with etcd certificates.
		etcdCertVolume := v1core.Volume{
			Name: "etcd-ssl-cert",
			VolumeSource: v1core.VolumeSource{
				Secret: &v1core.SecretVolumeSource{
					SecretName: "lcm-secrets",
					Items: []v1core.KeyToPath{
						v1core.KeyToPath{
							Key:  "DLAAS_ETCD_CERT",
							Path: "etcd/etcd.cert",
						},
					},
				},
			},
		}

		// Shared between the learner and helper.
		jobVolume := volumeSpecs[learnerID-1]

		// Each container mounts this volume.
		jobMount := v1core.VolumeMount{
			Name:      jobVolume.Name,
			MountPath: "/job",
			SubPath:   fmt.Sprintf("%s_learner-%d", req.TrainingId, learnerID),
		}

		envVars := populateLearnerEnvVariablesAndLabels(req, constructPSName(req.Name), getServiceNames(serviceSpecs), numLearners, learnerID, useNativeDistribution, logr)
		learnerTag := getLearnerTag(req, logr)

		// Learner deployment
		learnerName := constructLearnerName(learnerID, req.Name)
		learnerSpec := getGpuDeploymentSpec(learnerName, req.TrainingId, req.Resources.Schedpolicy, logr)
		falseVar := false
		learnerSpec.Spec.Template.Spec.AutomountServiceAccountToken = &falseVar
		learnerSpec.Spec.Template.Spec.Volumes = append(learnerSpec.Spec.Template.Spec.Volumes, *jobVolume)
		learnerSpec.Spec.Template.Spec.Containers = []v1core.Container{
			constructLearnerContainer(req, learnerID, learnerTag, envVars, jobMount, logr, mountTrainingDataStoreInLearner, mountResultsStoreInLearner, mountSSHCertsAsSecrets),
		}
		setServiceTypeLabel(&learnerSpec.Spec.Template.ObjectMeta, "dlaas-learner")

		// Helper deployment
		helperContainers := []v1core.Container{
			constructControllerContainer(learnerNodeBasePath, learnerNodeStatusPath, summaryMetricsPath, jobBasePath(req.TrainingId), jobMount, etcdCertVolume.Name, logr, mountTrainingDataStoreInLearner, mountResultsStoreInLearner),
			constructLoadModelContainer(jobMount, envVars),
			constructLogCollector(jobMount, s.k8sClient, req, learnerID, envVars, logr),
		}
		if mountTrainingDataStoreInLearner {
			region := req.EnvVars["DATA_STORE_REGION"]
			if region == "" {
				region = "us-standard"
			}
			cosInputVolume := v1core.Volume{
				Name: "cosinputmount-" + req.Name,
				VolumeSource: v1core.VolumeSource{
					FlexVolume: &v1core.FlexVolumeSource{
						Driver:    cosMountDriverName,
						FSType:    "",
						SecretRef: &v1core.LocalObjectReference{Name: trainingMountSecretName},
						ReadOnly:  true,
						Options: map[string]string{
							"bucket":   req.EnvVars["DATA_STORE_OBJECTID"],
							"endpoint": req.EnvVars["DATA_STORE_AUTHURL"],
							"region":   region,
							// We take over the resources taken by the following containers, which are not loaded when cos_mount is used, plus a bit more
							// load-data: 2GB memory, 1CPU, store-results: 512MB memory, 0.5 CPU, store-logs: 512MB memory, 0.5 CPU
							"cache-size-gb":  "6",  // should be a multiple of expected file size * number of application prefetch threads
							"chunk-size-mb":  "52", // value suggested for cruiser10 by benchmarking with a dallas COS instance
							"parallel-count": "5",  // should be at least expected file size / chunk size.  Extra threads will just sit idle
							"ensure-disk-free": "2048", // don't completely fill the cache, leave some buffer for parallel thread pulls
							"tls-cipher-suite": "AES256-GCM-SHA384",
							"multireq-max": "20",
							"stat-cache-size": "100000",
							"debug-level": "warn",
							"curl-debug": "false",
						},
					},
				},
			}
			learnerSpec.Spec.Template.Spec.Volumes = append(learnerSpec.Spec.Template.Spec.Volumes, cosInputVolume)
		} else {
			helperContainers = append(helperContainers, constructLoadTrainingDataContainer(jobMount, envVars))
		}

		if mountResultsStoreInLearner {
			region := req.EnvVars["RESULT_STORE_REGION"]
			if region == "" {
				region = "us-standard"
			}
			// RESULT_STORE_OBJECTID has the trainingId appended to the end of it.  We just want the bucket name.  Cut off "/TrainingId"
			bucketname := req.EnvVars["RESULT_STORE_OBJECTID"][0 : len(req.EnvVars["RESULT_STORE_OBJECTID"])-len(req.TrainingId)-1]
			cosOutputVolume := v1core.Volume{
				Name: "cosoutputmount-" + req.Name,
				VolumeSource: v1core.VolumeSource{
					FlexVolume: &v1core.FlexVolumeSource{
						Driver:    cosMountDriverName,
						FSType:    "",
						SecretRef: &v1core.LocalObjectReference{Name: resultsMountSecretName},
						ReadOnly:  false,
						Options: map[string]string{
							"bucket":   bucketname,
							"endpoint": req.EnvVars["RESULT_STORE_AUTHURL"],
							"region":   region,
							// tuning values suitable for writing checkpoints and logs
							"cache-size-gb":  "0",
							"chunk-size-mb":  "52",
							"parallel-count": "2",
							"ensure-disk-free": "2048",
							"tls-cipher-suite": "AES256-GCM-SHA384",
							"multireq-max": "20",
							"stat-cache-size": "100000",
							"debug-level": "warn",
							"curl-debug": "false",
						},
					},
				},
			}
			learnerSpec.Spec.Template.Spec.Volumes = append(learnerSpec.Spec.Template.Spec.Volumes, cosOutputVolume)
		} else {
			helperContainers = append(helperContainers,
				constructStoreResultsContainer(jobMount, learnerID, envVars),
				constructStoreLogsContainer(jobMount, learnerID, envVars))
		}

		//defining SSH cert as volume
		if mountSSHCertsAsSecrets {
			var permissions int32
			permissions = 0400
			sshCertVolume := v1core.Volume{
				Name: "sshcertmount-" + req.Name,
				VolumeSource: v1core.VolumeSource{
					Secret: &v1core.SecretVolumeSource{
						SecretName:  sshSecret.Name,
						DefaultMode: &permissions,
					},
				},
			}
			learnerSpec.Spec.Template.Spec.Volumes = append(learnerSpec.Spec.Template.Spec.Volumes, sshCertVolume)
		}

		if useSplitLearner {
			helperName := constructLearnerHelperName(learnerID, req.Name)
			helperSpec := getGpuDeploymentSpec(helperName, req.TrainingId, req.Resources.Schedpolicy, logr)
			helperSpec.Spec.Template.Spec.Volumes = append(helperSpec.Spec.Template.Spec.Volumes, etcdCertVolume, *jobVolume)
			helperSpec.Spec.Template.Spec.Containers = helperContainers
			setServiceTypeLabel(&helperSpec.Spec.Template.ObjectMeta, "dlaas-lhelper")
			deploymentSpecs = append(deploymentSpecs, helperSpec)
		} else {
			setServiceTypeLabel(&learnerSpec.Spec.Template.ObjectMeta, "dlaas-learnerandhelper")
			learnerSpec.Spec.Template.Spec.Volumes = append(learnerSpec.Spec.Template.Spec.Volumes, etcdCertVolume)
			for _, c := range helperContainers {
				learnerSpec.Spec.Template.Spec.Containers = append(learnerSpec.Spec.Template.Spec.Containers, c)
			}
		}

		extendLearnerDeployment(learnerSpec)
		deploymentSpecs = append(deploymentSpecs, learnerSpec)
	}

	return serviceSpecs, volumeClaimSpecs, deploymentSpecs, secretSpecs, nil
}

// Get the spec for deployments that need to run on a GPU node in Armada.
func getGpuDeploymentSpec(name string, trainingID string, schedulePolicy string, logr *logger.LocLoggingEntry) *v1beta1.Deployment {
	annotations := make(map[string]string)

	// This toleration is needed to get scheduled with the GPU-tainted nodes on the Armada cluster.
	annotations["scheduler.alpha.kubernetes.io/tolerations"] =
		`[ { "key": "dedicated", "operator": "Equal", "value": "gpu-task" } ]`

	if schedulePolicy == "spread" {
		// This annotation spreads the workloads on the cluster based on free GPUs on the Armada cluster.
		annotations["scheduler.alpha.kubernetes.io/nvidiaGPU"] = `{ "AllocationPriority": "Spread" }`
		logr.Debugf("User request to spread the workload in the cluser %s", schedulePolicy)
	} else {
		// This annotation packs the workloads on the cluster based on free GPUs on the Armada cluster.
		// This is the default policy
		annotations["scheduler.alpha.kubernetes.io/nvidiaGPU"] = `{ "AllocationPriority": "Dense" }`
		logr.Debugf("Adding default pack scheduling policy: %s", schedulePolicy)
	}

	labels := map[string]string{
		"app":         name,
		"training_id": trainingID,
	}

	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)

	deploySpec := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.DeploymentSpec{
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &v1beta1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(0),
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: int32(1),
					},
				},
			},
			Template: v1core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: v1core.PodSpec{
					Containers:    nil, // will be filled in later
					RestartPolicy: v1core.RestartPolicyAlways,
					DNSPolicy:     v1core.DNSClusterFirst,
					Volumes:       []v1core.Volume{},
					ImagePullSecrets: []v1core.LocalObjectReference{
						v1core.LocalObjectReference{
							Name: imagePullSecret,
						},
					},
					Tolerations: []v1core.Toleration{
						v1core.Toleration{
							Key:      "dedicated",
							Operator: v1core.TolerationOpEqual,
							Value:    "gpu-task",
							Effect:   v1core.TaintEffectNoSchedule,
						},
					},
				},
			},
		},
	}

	return deploySpec
}

func getLearnerImageTagInRequest(req *service.JobDeploymentRequest) string {
	for k, v := range req.EnvVars {
		if k == "DLAAS_LEARNER_IMAGE_TAG" {
			return v
		}
	}
	return ""
}

func getLearnerTag(req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) string {
	imageName := fmt.Sprintf("%s_gpu_%s", req.Framework, req.Version)

	// default value will be default learner tag key
	learnerTag := viper.GetString(config.LearnerTagKey)
	logr.Debugf("(LCM getLeanerTag) learnerTag (default): %s", learnerTag)

	// Use any tag in the request (ie, specified in the manifest)
	learnerImageTagInManifest := getLearnerImageTagInRequest(req)
	if "" == learnerImageTagInManifest {
		// not in request; try looking up from configmap/learner-config
		learnerConfigFile := config.GetCurrentLearnerConfigLocationFromCombination(imageName)
		if "" != learnerConfigFile {
			b, err := ioutil.ReadFile(learnerConfigFile)
			if err == nil {
				learnerTag = string(b)
				logr.Debugf("(LCM getLearnerTag) learnerTag (from %s): %s", learnerConfigFile, learnerTag)
			}
		}
	} else {
		learnerTag = learnerImageTagInManifest
	}

	return learnerTag
}

// Deploy learners and helpers of a specific training job
func deployLearnersAndHelpers(ctx context.Context, s *lcmService, req *service.JobDeploymentRequest, useNativeDistribution bool) error {
	logr := logger.LocLogger(s.logWithJobDeploymentRequest(req))

	serviceSpecs, volumeClaimSpecs, deploySpecs, secretSpecs, err := defineLearnerAndHelperObjects(s, req, useNativeDistribution, logr)
	if err != nil {
		logr.WithError(err).Errorf("Failed to create the k8s spec for training %s", req.TrainingId)
		return err
	}

	logr.Debugf("in deployLearnersAndHelpers, #services: %d", len(serviceSpecs))
	logr.Debugf("in deployLearnersAndHelpers, #claims: %d", len(volumeClaimSpecs))
	logr.Debugf("in deployLearnersAndHelpers, #deployments: %d", len(deploySpecs))
	// don't print secrets.  They're secret :-)

	// create secrets
	for _, spec := range secretSpecs {
		err := util.Retry(10, 10*time.Second, "CreateSecret", logr, func() error {
			secret, err := s.k8sClient.Core().Secrets(config.GetLearnerNamespace()).Create(spec)
			if err != nil {
				logr.WithError(err).Errorf("Retrying after failure to create Secret: %s\n", spec.ObjectMeta.Name)
				return err
			}
			logr.Infof("Successfully created Secret: %s", secret.ObjectMeta.Name)
			return nil
		})
		if err != nil {
			logr.WithError(err).Errorf("Failed to create Secret after trying multiple times: %s\n", spec.ObjectMeta.Name)
			return err
		}
	}

	// Create the services.
	for _, spec := range serviceSpecs {
		err := util.Retry(10, 10*time.Second, "CreateService", logr, func() error {
			service, err := s.k8sClient.Core().Services(config.GetLearnerNamespace()).Create(spec)
			if err != nil {
				logr.WithError(err).Errorf("Retrying after failure to create service: %s\n", spec.ObjectMeta.Name)
				return err
			}
			logr.Infof("Successfully created service: %s", service.ObjectMeta.Name)
			return nil
		})
		if err != nil {
			logr.WithError(err).Errorf("Failed to create service after trying multiple times: %s\n", spec.ObjectMeta.Name)
			return err
		}
	}

	// Create the volume claims.
	for _, spec := range volumeClaimSpecs {
		err := util.Retry(10, 10*time.Second, "CreateVolumeClaim", logr, func() error {
			claim, err := s.k8sClient.Core().PersistentVolumeClaims(config.GetLearnerNamespace()).Create(spec)
			if err != nil {
				logr.WithError(err).Errorf("Retrying after failure to create volume claim: %s\n", spec.ObjectMeta.Name)
				return err
			}
			logr.Infof("Successfully created volume claim: %s", claim.ObjectMeta.Name)
			return nil
		})
		if err != nil {
			logr.WithError(err).Errorf("Failed to create volume claim after trying multiple times: %s\n", spec.ObjectMeta.Name)
			return err
		}
	}

	// Create the deployments.
	for _, spec := range deploySpecs {
		err := util.Retry(10, 10*time.Second, "CreateLearnerDeployment", logr, func() error {
			deployment, err := s.k8sClient.Extensions().Deployments(config.GetLearnerNamespace()).Create(spec)
			if err != nil {
				logr.WithError(err).Errorf("Retrying after failure to create deployment: %s\n", spec.ObjectMeta.Name)
				return err
			}
			logr.Infof("Successfully created learner: %s", deployment.ObjectMeta.Name)
			return nil
		})
		if err != nil {
			logr.WithError(err).Errorf("Failed to create deployment after trying multiple times: %s\n", spec.ObjectMeta.Name)
			return err
		}
	}

	return nil
}
