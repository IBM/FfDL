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
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/lcm/service/lcm/helper"
	"github.com/IBM/FfDL/lcm/service/lcm/learner"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"

	"golang.org/x/net/context"

	"k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// PodLevelJobDir represents the place to store the job state indicator files,
// as well as the $BREAK_FILE and $EXITCODE_FILE.
const PodLevelJobDir = "/job"
//const PodLevelJobDir = "/nfs/var/nfs/general/"

// PodLevelLogDir represents the place to store the per-learner logs.
const PodLevelLogDir = PodLevelJobDir + "/logs"

const doLearnerDeploymentDebuggingDiagnostics = false

//Training ...
type Training interface {
	Start() error
	//these can be implemented as a part of training
	//Halt() error
	//Stop() error
}

type training struct {
	ctx        context.Context
	k8sClient  kubernetes.Interface
	req        *service.JobDeploymentRequest
	trainingID string
	learner    learnerDefinition
	helper     helperDefinition
	logr       *logger.LocLoggingEntry
}

type splitTrainingBOM struct {
	secrets              []*v1core.Secret
	service              *v1core.Service
	sharedVolumeClaimBOM *v1core.PersistentVolumeClaim
	learnerBOM           *v1beta1.StatefulSet
	helperBOM            *v1beta1.Deployment
	numLearners          int
}

type nonSplitTrainingBOM struct {
	secrets     []*v1core.Secret
	service     *v1core.Service
	learnerBOM  *v1beta1.StatefulSet
	numLearners int
}

type splitTraining struct {
	*training
}

type nonSplitTraining struct {
	*training
}

type learnerDefinition struct {
	secrets                                                     []*v1core.Secret
	volumes                                                     []v1core.Volume
	volumeMounts                                                []v1core.VolumeMount
	envVars                                                     []v1core.EnvVar
	mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool
	numberOfLearners                                            int
	name                                                        string
}

type helperDefinition struct {
	sharedVolume      v1core.Volume
	etcdVolume        v1core.Volume
	etcdVolumeMount   v1core.VolumeMount
	sharedVolumeMount v1core.VolumeMount
	sharedEnvVars     []v1core.EnvVar
	sharedVolumeClaim *v1core.PersistentVolumeClaim
	name              string
}

//NewTraining ...
func NewTraining(ctx context.Context, k8sClient kubernetes.Interface, req *service.JobDeploymentRequest,
	log *logger.LocLoggingEntry) Training {

	log.Debugf("DATA_STORE_TYPE: %s, RESULT_STORE_TYPE: %s, MODEL_STORE_TYPE: %s",
		req.EnvVars["DATA_STORE_TYPE"],
		req.EnvVars["RESULT_STORE_TYPE"],
		req.EnvVars["MODEL_STORE_TYPE"])

	learnerName := fmt.Sprintf("learner-%s", req.Name)
	helperName := fmt.Sprintf("lhelper-%s", req.Name)
	numLearners := int(req.GetResources().Learners)
	if numLearners < 1 {
		numLearners = 1
	}

	dataStoreType := req.EnvVars["DATA_STORE_TYPE"]
	resultStoreStype := req.EnvVars["RESULT_STORE_TYPE"]
	mountTrainingDataStoreInLearner := !(dataStoreType == learner.DataStoreTypeS3)
	mountResultsStoreInLearner := !(resultStoreStype == learner.DataStoreTypeS3)


	logr := log.WithFields(logrus.Fields{
		"learner_name": learnerName,
		"helper_name":  helperName,
		"mounted_cos":  mountResultsStoreInLearner && mountTrainingDataStoreInLearner,
	})

	envVarsFromDeploymentRequest := extractEnvVarsFromDeploymentRequest(req) //shared across all containers of training
	envvarsForLearner := envVarsForDeployingLearner(envVarsFromDeploymentRequest, req.TrainingId,
		numLearners, learnerName, mountTrainingDataStoreInLearner, mountResultsStoreInLearner) //only for learner

	learnerVolumes := volumesForLearner(req, envvarsForLearner, mountTrainingDataStoreInLearner, mountResultsStoreInLearner, logr)
	learnerVolumeSpecs := learnerVolumes.CreateVolumeForLearner()
	learnerVolumeSpecs = extendLearnerVolumes(learnerVolumeSpecs, logr)
	learnerDefn := learnerDefinition{
		secrets:                         secretsForDeployingLearner(req, mountTrainingDataStoreInLearner, mountResultsStoreInLearner),
		volumes:                         learnerVolumeSpecs,
		volumeMounts:                    learnerVolumes.CreateVolumeMountsForLearner(),
		envVars:                         envvarsForLearner,
		numberOfLearners:                numLearners,
		mountTrainingDataStoreInLearner: mountTrainingDataStoreInLearner,
		mountResultsStoreInLearner:      mountResultsStoreInLearner,
		name: learnerName,
	}

	helperVolumes := volumesForHelper(req, logr)
	helperDefn := helperDefinition{
		etcdVolume:        helperVolumes.CreateETCDVolume(),
		etcdVolumeMount:   helperVolumes.CreateETCDVolumeMount(),
		sharedEnvVars:     envVarsFromDeploymentRequest,
		sharedVolume:      helperVolumes.CreateDataVolume(),
		sharedVolumeMount: helperVolumes.CreateDataVolumeMount(),
		sharedVolumeClaim: helperVolumes.DynamicPVCReference(),
		name:              helperName,
	}
	logr.Debugf("sharedVolume: %v+", helperDefn.sharedVolume)
	logr.Debugf("sharedVolumeMount: %v+", helperDefn.sharedVolumeMount)
	logr.Debugf("sharedVolumeClaim: %v+", helperDefn.sharedVolumeClaim)

	logr.Debugf("sharedVolume: %v+", helperDefn.sharedVolume)
	logr.Debugf("sharedVolumeMount: %v+", helperDefn.sharedVolumeMount)
	logr.Debugf("sharedVolumeClaim: %v+", helperDefn.sharedVolumeClaim)

	if helperVolumes.SharedNonSplitLearnerHelperVolume != nil {
		//this should not be the default case, we should be running in split mode by default
		logr.Warnf("starting deploying learner infra for non split learning, this is not expected")
		return nonSplitTraining{&training{ctx, k8sClient, req, req.TrainingId,
			learnerDefn, helperDefn, logr}}
	}
	logr.Infof("starting deploying learner infra for split learning")
	return splitTraining{&training{ctx, k8sClient, req, req.TrainingId,
		learnerDefn, helperDefn, logr}}
}

///-------

func secretsForDeployingLearner(req *service.JobDeploymentRequest, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []*v1core.Secret {
	//irrespective of split/non split learners these secrets need to be created

	secretsStruct := learner.Secrets{}

	if mountTrainingDataStoreInLearner {
		trainingMountSecretName := "cossecretdata-" + req.Name
		secretsStruct.TrainingDataSecret = &learner.COSVolumeSecret{ID: trainingMountSecretName, TrainingID: req.TrainingId, Username: req.EnvVars["DATA_STORE_USERNAME"], APIKey: req.EnvVars["DATA_STORE_APIKEY"]}
	}

	if mountResultsStoreInLearner {
		resultsMountSecretName := "cossecretresults-" + req.Name
		secretsStruct.ResultsDirSecret = &learner.COSVolumeSecret{ID: resultsMountSecretName, TrainingID: req.TrainingId, Username: req.EnvVars["RESULT_STORE_USERNAME"], APIKey: req.EnvVars["RESULT_STORE_APIKEY"]}
	}
	secretSpecs := learner.CreateVolumeSecretsSpec(secretsStruct)

	return secretSpecs
}

func dumpValues(label string, envVars []v1core.EnvVar) {
	for i, ev := range envVars {
		fmt.Printf("%s[%d]: %s: %s\n", label, i, ev.Name, ev.Value)
	}
}

func volumesForLearner(req *service.JobDeploymentRequest, learnerEnvVars []v1core.EnvVar,
	mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool, logr *logger.LocLoggingEntry) learner.Volumes {

	volumesStruct := learner.Volumes{}

	if doLearnerDeploymentDebuggingDiagnostics {
		dumpValues("learnerEnvVars", learnerEnvVars)
		for k, v := range req.EnvVars {
			fmt.Printf("req.EnvVars: key[%s] value[%s]\n", k, v)
		}
	}

	var region = "us-standard"

	if mountTrainingDataStoreInLearner {
		dataStoreType := req.EnvVars["DATA_STORE_TYPE"]
		logr.Debugf("dataStoreType: %s", dataStoreType)
		if dataStoreType == learner.DataStoreTypeMountCOSS3 {
			region = req.EnvVars["DATA_STORE_REGION"]
			if region == "" {
				region = "us-standard"
			}
			configValStr := config.GetString("MOUNTCOS_GB_CACHE_PER_GPU")
			cacheSize, err := strconv.Atoi(configValStr)
			if err != nil {
				cacheSize = 6
				logr.Warnf("DLAAS_MOUNTCOS_GB_CACHE_PER_GPU value %s is not an integer.  Defaulting to %dGB/GPU",
					configValStr, cacheSize)
			}
			cacheSize = cacheSize * int(req.Resources.Gpus)
			// reserve 1/3 of cache for prefetching, up to a limit (diskFree is specified in MB, cache in GB)
			diskFree := (cacheSize * 1024) / 3
			if diskFree > 10000 {
				diskFree = 10000
			}

			volumesStruct.TrainingData = &learner.COSVolume{
				VolumeType: dataStoreType,
				ID: "cosinputmount-" + req.Name,
				Region: region,
				Bucket: req.EnvVars["DATA_STORE_OBJECTID"],
				Endpoint: req.EnvVars["DATA_STORE_AUTHURL"],
				SecretRef: "cossecretdata-" + req.Name,
				MountSpec: learner.VolumeMountSpec {
					MountPath: getValue(learnerEnvVars, "DATA_DIR"),
					SubPath: "",
				},
				CacheSize: strconv.Itoa(cacheSize),
				DiskFree: strconv.Itoa(diskFree),
			}
		} else if dataStoreType == learner.DataStoreHostMountVolume {
			hostPath := req.EnvVars["DATA_STORE_PATH"]
			// The variable the learner will get is the concatenated mount path from envvars.go
			mountPath := getValue(learnerEnvVars, "DATA_DIR")
			// While this is the unadulterated value
			subPath := req.EnvVars["DATA_DIR"]
			fmt.Printf("(data) hostPath=%s\nmountPath=%s\nsubPath=%s\n", hostPath, mountPath, subPath)
			volumesStruct.TrainingData = &learner.COSVolume{
				VolumeType: dataStoreType,
				ID: "inputmount-" + req.Name,
				HostPath: hostPath,

				MountSpec: learner.VolumeMountSpec {
					Name: "inputmount-" + req.Name,
					MountPath: mountPath,
					SubPath: subPath,
				},
			}
			logr.Debugf("TrainingData volume request: %v+", volumesStruct.TrainingData)
		}
	}
	if mountResultsStoreInLearner {
		resultStoreType := req.EnvVars["RESULT_STORE_TYPE"]
		logr.Debugf("resultStoreType: %s", resultStoreType)
		if resultStoreType == learner.DataStoreTypeMountCOSS3 {
			region = req.EnvVars["RESULT_STORE_REGION"]
			if region == "" {
				region = "us-standard"
			}
			// RESULT_STORE_OBJECTID has the trainingId appended to the end of it.
			// We just want the bucket name.  Cut off "/TrainingId"
			resultStoreBucket :=
				req.EnvVars["RESULT_STORE_OBJECTID"][0: len(req.EnvVars["RESULT_STORE_OBJECTID"])-len(req.TrainingId)-1]
			volumesStruct.ResultsDir = &learner.COSVolume{
				VolumeType: resultStoreType,
				ID:        "cosoutputmount-" + req.Name,
				Region:    region,
				Bucket:    resultStoreBucket,
				Endpoint:  req.EnvVars["RESULT_STORE_AUTHURL"],
				SecretRef: "cossecretresults-" + req.Name,
				MountSpec: learner.VolumeMountSpec{
					MountPath: getValue(learnerEnvVars, "RESULT_DIR"),
					SubPath:   "",
				},
				CacheSize: "0",
				DiskFree:  "2048",
			}
		} else if resultStoreType == learner.DataStoreHostMountVolume {
			hostPath := req.EnvVars["RESULT_STORE_PATH"]
			// The variable the learner will get is the concatenated mount path from envvars.go
			mountPath := getValue(learnerEnvVars, "RESULT_DIR")
			// While this is the unadulterated value
			subPath := req.EnvVars["RESULT_DIR"]

			fmt.Printf("(result) hostPath=%s\nmountPath=%s\nsubPath=%s\n", hostPath, mountPath, subPath)
			volumesStruct.ResultsDir = &learner.COSVolume{
				VolumeType: resultStoreType,
				ID: "outputmount-" + req.Name,
				HostPath: hostPath,

				MountSpec: learner.VolumeMountSpec {
					Name: "outputmount-" + req.Name,
					MountPath: mountPath,
					SubPath: subPath,
				},
			}
			logr.Debugf("ResultsDir volume request: %v+", volumesStruct.ResultsDir)
		}
	}

	return volumesStruct
}

func volumesForHelper(req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) helper.Volumes {
	volumesStruct := helper.Volumes{}

	volumesStruct.ETCDVolume = &helper.ETCDVolume{Name: "etcd-ssl-cert"}

	volumeSize := getStorageSize(req.Resources)
	logr.Debugf("Requested storage for job of size %d bytes", volumeSize)
	useDynamicExternalVolume := volumeSize > 0

	staticVolumeName := getStaticVolume(logr)
	logr.Debugf("Static volume for job is %s", staticVolumeName)
	useStaticExternalVolume := len(staticVolumeName) > 0

	useSplitLearner := useDynamicExternalVolume || useStaticExternalVolume

	logr.Debugf("useSplitLearner is %t", useSplitLearner)

	if !useSplitLearner {
		logr.Infof("Starting training %s with NON SPLIT MODE %d", req.TrainingId, volumeSize)
		volumesStruct.SharedNonSplitLearnerHelperVolume = &helper.LocalVolume{Name: "jobdata",
			MountSpec: helper.VolumeMountSpec{MountPath: PodLevelJobDir, SubPath: req.TrainingId}}
	}

	if useStaticExternalVolume {
		logr.Infof("Using static external volume for training %s with name %s", req.TrainingId, staticVolumeName)
		volumesStruct.SharedSplitLearnerHelperVolume = &helper.SharedNFSVolume{
			Name: "jobdata",
			PVCClaimName: staticVolumeName,
			PVC: nil,
			MountSpec: helper.VolumeMountSpec{
				MountPath: PodLevelJobDir,
				SubPath: req.TrainingId,
			},
		}

	} else if useDynamicExternalVolume {
		logr.Infof("Using dynamic external volume...")
		sharedVolumeClaim := constructVolumeClaim(req.Name, config.GetLearnerNamespace(),
			volumeSize, map[string]string{"training_id": req.TrainingId}, logr)
		logr.Infof("Using dynamic external volume for Training %s with name %s",
			req.TrainingId, sharedVolumeClaim.Name)
		volumesStruct.SharedSplitLearnerHelperVolume = &helper.SharedNFSVolume{
			Name: "jobdata",
			PVCClaimName: sharedVolumeClaim.Name,
			PVC: sharedVolumeClaim,
			MountSpec: helper.VolumeMountSpec{
				MountPath: PodLevelJobDir,
				SubPath: req.TrainingId,
			},
		}
	}
	return volumesStruct
}

//list of shared env vars shared by all containers in helper and learner pod
func extractEnvVarsFromDeploymentRequest(req *service.JobDeploymentRequest) []v1core.EnvVar {
	var envVars []v1core.EnvVar
	for k, v := range req.EnvVars {
		envVars = append(envVars, v1core.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return envVars
}

//needs to happen before the volume creation since we muck around with the value and change the paths of data/result dir
func envVarsForDeployingLearner(existingEnvVars []v1core.EnvVar, trainingID string, numLearners int, statefulsetName string, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []v1core.EnvVar {
	return learner.PopulateLearnerEnvVariablesAndLabels(existingEnvVars, trainingID, numLearners, statefulsetName, mountTrainingDataStoreInLearner, mountResultsStoreInLearner)

}

func (t *training) constructAuxillaryContainers() []v1core.Container {
	learnerDefn := t.learner
	helperDefn := t.helper
	helperContainers := []v1core.Container{
		constructControllerContainer(t.req.TrainingId, helperDefn.etcdVolumeMount, helperDefn.sharedVolumeMount, learnerDefn.mountTrainingDataStoreInLearner, learnerDefn.mountResultsStoreInLearner),
		constructLoadModelContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars),
		constructLogCollector(helperDefn.sharedVolumeMount, t.k8sClient, t.req, helperDefn.sharedEnvVars, t.logr),
	}

	if !learnerDefn.mountTrainingDataStoreInLearner {
		helperContainers = append(helperContainers, constructLoadTrainingDataContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
	}
	if !learnerDefn.mountResultsStoreInLearner {
		helperContainers = append(helperContainers, constructStoreResultsContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
		helperContainers = append(helperContainers, constructStoreLogsContainer(helperDefn.sharedVolumeMount, helperDefn.sharedEnvVars))
	}
	return helperContainers
}
