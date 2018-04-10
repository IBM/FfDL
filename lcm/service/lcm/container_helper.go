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
	"path"
	"strings"
	"text/template"
	"os"
	"strconv"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/service"

	"github.com/spf13/viper"

	"bytes"

	"github.com/IBM/FfDL/commons/logger"
	"gopkg.in/yaml.v2"
	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const logCollectorContainerName string = "log-collector" // the name of the learner container in the pod
const controllerContainerName = "controller"
const loadDataContainerName = "load-data"
const loadModelContainerName = "load-model"
const learnerContainerName = "learner"
const storeResultsContainerName = "store-results"
const storeLogsContainerName = "store-logs"
const learnerConfigDir = "/etc/learner-config"

const simpleLogCollectorName = "log_collector"

const logCollectorBadTagNoTDSFound = "dummy-tag-no-tds-found"

// The whitelisted environment variables to pass to the learner container.
// Use a map structure for O(1) lookups.
var learnerContainerEnvVars = map[string]struct{}{
	"MODEL_DIR":        {},
	"DATA_DIR":         {},
	"RESULT_DIR":       {},
	"LOG_DIR":          {},
	"TRAINING_JOB":     {},
	"TRAINING_COMMAND": {},
	"TRAINING_ID":      {},
	"LEARNER_ID":       {},
	"GPU_COUNT":        {},
	"NUM_LEARNERS":     {},
	"SERVICE_NAME_1":   {},
	"SERVICE_NAME_2":   {},
	"SERVICE_NAME_3":   {},
	"SERVICE_NAME_4":   {},
	"SERVICE_NAME_5":   {},
	"SERVICE_NAME_6":   {},
	"SERVICE_NAME_7":   {},
	"SERVICE_NAME_8":   {},
	"SERVICE_NAME_9":   {},
	"SERVICE_NAME_10":  {},
	"SERVICE_NAME_11":  {},
	"SERVICE_NAME_12":  {},
	"SERVICE_NAME_13":  {},
	"SERVICE_NAME_14":  {},
	"SERVICE_NAME_15":  {},
	"SERVICE_NAME_16":  {},
}

// TODO make configurablee
var defaultPullPolicy = v1core.PullIfNotPresent

// valid names of databroker types that map to "databroker_<type>" Docker image names
var validDatabrokerTypes = []string{"objectstorage", "s3"}

// default databroker type
var defaultDatabrokerType = "objectstorage"

const (
	workerPort int32 = 2222
	sshPort    int32 = 22
)

// PodLevelJobDir represents the place to store the job state indicator files,
// as well as the $BREAK_FILE and $EXITCODE_FILE.
const PodLevelJobDir = "/job"

// PodLevelLogDir represents the place to store the per-learner logs.
const PodLevelLogDir = PodLevelJobDir + "/logs"

// Helper pods spec
var (
	storeResultsMilliCPU = getHelperSpec("storeResultsMilliCPU")
	storeResultsMemInMB = getHelperSpec("storeResultsMemInMB")
	loadModelMilliCPU = getHelperSpec("loadModelMilliCPU")
	loadModelMemInMB = getHelperSpec("loadModelMemInMB")
	loadTrainingDataMilliCPU = getHelperSpec("loadTrainingDataMilliCPU")
	loadTrainingDataMemInMB = getHelperSpec("loadTrainingDataMemInMB")
	logCollectorMilliCPU = getHelperSpec("logCollectorMilliCPU")
	logCollectorMemInMB = getHelperSpec("logCollectorMemInMB")
	controllerMilliCPU = getHelperSpec("controllerMilliCPU")
	controllerMemInMB = getHelperSpec("controllerMemInMB")
)

func getHelperSpec(specName string) int {
	result, err := strconv.Atoi(os.Getenv(specName))
	if err != nil {
		return 100 // Default spec for helper pod
	}
	return result
}

func constructControllerContainer(learnerNodeBasePath, learnerNodeStatusPath, summaryMetricsPath, jobBasePath string, jobVolumeMount v1core.VolumeMount, etcdVolumeName string, logr *logger.LocLoggingEntry, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) v1core.Container {

	var getEnvVarFromLCMSecret = func(lookupkey string) v1core.EnvVar {
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

	mountPath := PodLevelJobDir
	logr.Debugf("constructControllerContainer: PodLevelJobDir=%s", mountPath)

	servicesTag := viper.GetString(config.ServicesTagKey)
	logr.Debugf("servicesTag: %s", servicesTag)

	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	controllerImageName := controllerImageNameExtended(dockerRegistry, servicesTag)
	logr.Debugf("controllerImageName: %s", controllerImageName)

	cmd := fmt.Sprintf("controller.sh")

	// short-circuit the load and store databrokers when we mount object storage directly
	if mountResultsStoreInLearner {
		cmd = "echo 0 > " + PodLevelJobDir + "/store-results.exit && " + "echo 0 > " + PodLevelJobDir + "/store-logs.exit && " + cmd
	}
	if mountTrainingDataStoreInLearner {
		cmd = "echo 0 > " + PodLevelJobDir + "/load-data.exit && " + cmd
	}

	cpuCount := v1resource.NewMilliQuantity(int64(controllerMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(controllerMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	container := v1core.Container{
		Name:    controllerContainerName,
		Image:   controllerImageName,
		Command: []string{"sh", "-c", cmd},
		Env: []v1core.EnvVar{
			v1core.EnvVar{Name: "JOB_STATE_DIR", Value: PodLevelJobDir},
			v1core.EnvVar{Name: "JOB_LEARNER_ZNODE_PATH", Value: learnerNodeBasePath},
			v1core.EnvVar{Name: "JOB_BASE_PATH", Value: jobBasePath},
			v1core.EnvVar{Name: "JOB_LEARNER_ZNODE_STATUS_PATH", Value: learnerNodeStatusPath},
			v1core.EnvVar{Name: "JOB_LEARNER_SUMMARY_STATS_PATH", Value: summaryMetricsPath},
			v1core.EnvVar{Name: "DOWNWARD_API_POD_NAME", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			v1core.EnvVar{Name: "DOWNWARD_API_POD_NAMESPACE", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
			getEnvVarFromLCMSecret("DLAAS_ETCD_ADDRESS"),
			getEnvVarFromLCMSecret("DLAAS_ETCD_USERNAME"),
			getEnvVarFromLCMSecret("DLAAS_ETCD_PASSWORD"),
			getEnvVarFromLCMSecret("DLAAS_ETCD_PREFIX"),
		},
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
		},
		VolumeMounts: []v1core.VolumeMount{
			jobVolumeMount,
			v1core.VolumeMount{
				Name:      etcdVolumeName,
				MountPath: "/etc/certs/",
				ReadOnly:  true,
			},
		},
		ImagePullPolicy: defaultPullPolicy,
	}
	return container
}

func fetchImageNameFromEvaluationMetrics(evalMetricsString string,
	learnerTag string,
	framework string,
	version string,
	logr *logger.LocLoggingEntry) (string, string) {

	logr.Debugf("evaluation_metrics: %v<end>", evalMetricsString)
	logCollectorImageShortName := simpleLogCollectorName

	learnerEMTag := learnerTag

	logr.Debugf("evalMetricsString: %s", evalMetricsString)
	if evalMetricsString != "" {
		em := make(map[interface{}]interface{})
		err := yaml.Unmarshal([]byte(evalMetricsString), &em)
		if err != nil {
			// Assuming pre-validation, this is unlikely to happen, so this is mostly a programmer assertion.
			logr.WithError(err).Error("evaluation_metrics was specified in manifest, but can't be parsed!")
		}

		m := em["evaluation_metrics"].(map[interface{}]interface{})

		if m != nil {
			val, ok := m["image_tag"]
			logr.Debugf("learner tag: %s %t", val, ok)
			if ok == false {
				// TODO: fix dropping underbar problem.  Somehow.
				// Having a hard time with, I think the yaml to string stuff, dropping underbars.
				val, ok = m["imagetag"]
			}
			if ok && val.(string) != "" {
				learnerEMTag = val.(string)
			}

			imageType, ok := m["type"]

			// Allow some synonyms for simple file extractor
			if ok && (imageType == "optivist" || imageType == "emetrics_file" || imageType == "file") {
				imageType = "emetrics_file_extractor"
			}

			if ok && imageType.(string) != "" {
				logr.Debugf("initial evaluation_metrics type: %s", imageType)
				// Assume the image name has been validated upstream
				logCollectorImageShortName = imageType.(string)
				if logCollectorImageShortName == "tensorboard" || logCollectorImageShortName == "tensorboard_extractor" {
					// TODO: Fix tensorflow/board docker image version matrix nightmare
					tensorBoardVersion := "1.3-py3"
					// if framework == tfFrameworkName {
					// 	tensorBoardVersion = version
					// }
					// Try to match the best version
					logCollectorImageShortName = fmt.Sprintf("%s_extract_%s", "tensorboard", tensorBoardVersion)
				}
				// Be flexible
				if logCollectorImageShortName == "null" || logCollectorImageShortName == "nil" ||
					logCollectorImageShortName == "logger" || logCollectorImageShortName == "none" {
					logCollectorImageShortName = simpleLogCollectorName
				}

			} else {
				logr.Error("evaluation_metrics type is empty")
				logCollectorImageShortName = simpleLogCollectorName
			}
		} else {
			logr.Debug("No evaluation metrics specified! (2)")
		}
	} else {
		logr.Debug("No evaluation metrics specified! (1)")
	}
	return logCollectorImageShortName, learnerEMTag
}

func findTrainingDataServiceTag(k8sClient kubernetes.Interface, logr *logger.LocLoggingEntry) string {
	selector := "service==ffdl-trainingdata"
	podInterface := k8sClient.Core().Pods(config.GetPodNamespace())
	pods, err := podInterface.List(metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		logr.WithError(err).Debugf("Could not find service=ffdl-trainingdata")
		// bad fallback, ideally should never happen
		return logCollectorBadTagNoTDSFound
	}
	nPods := len(pods.Items)
	if nPods > 0 {
		for i := nPods - 1; i >= 0; i-- {
			containerStatuses := pods.Items[i].Status.ContainerStatuses
			for _, containerStatus := range containerStatuses {
				imageName := containerStatus.Image
				// No tag, don't build a log-collector
				splits := strings.SplitAfter(imageName, ":")
				if splits != nil && len(splits) > 1 {
					return splits[len(splits)-1]
				}
			}
		}
	}
	// bad fallback, ideally should never happen
	return logCollectorBadTagNoTDSFound
}

func constructLogCollector(jobVolumeMount v1core.VolumeMount, k8sClient kubernetes.Interface, req *service.JobDeploymentRequest,
	learnerID int, envVars []v1core.EnvVar, logr *logger.LocLoggingEntry) v1core.Container {

	etcdEnvVars := populateETCDEnvVariables(req.TrainingId)
	envVarsWithEtcdVars := append(envVars, etcdEnvVars...)

	servicesTag := viper.GetString(config.ServicesTagKey)
	logr.Debugf("servicesTag: %s", servicesTag)

	mountPath := PodLevelJobDir

	defaultTag := findTrainingDataServiceTag(k8sClient, logr)
	logr.Debugf("default log-collector tag: " + defaultTag)

	logCollectorImageShortName, learnerEMTag := fetchImageNameFromEvaluationMetrics(req.EvaluationMetricsSpec,
		defaultTag, req.Framework, req.Version, logr)

	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	logCollectorImage :=
		fmt.Sprintf("%s/%s:%s", dockerRegistry, logCollectorImageShortName, learnerEMTag)

	// logCollectorImage := fmt.Sprintf("%s/log-collector:%s", dockerRegistry, servicesTag)
	logr.Debugf("logCollectorImage: %s", logCollectorImage)

	vars := make([]v1core.EnvVar, 0, len(envVarsWithEtcdVars))
	for _, ev := range envVarsWithEtcdVars {
		if strings.HasSuffix(ev.Name, "_DIR") {
			// Adjust the paths to be in the mount point.
			dir := path.Join(mountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: ev.Name, Value: dir})
			logr.Debugf("constructWaitingLearnerContainer, binding env var: %s=%s", ev.Name, dir)
		} else {
			vars = append(vars, ev)
		}
	}

	vars = append(vars, v1core.EnvVar{Name: "JOB_STATE_DIR", Value: PodLevelJobDir})
	vars = append(vars, v1core.EnvVar{Name: "TRAINING_DATA_NAMESPACE", Value: config.GetPodNamespace()})

	if req.EvaluationMetricsSpec != "" {
		vars = append(vars, v1core.EnvVar{Name: "EM_DESCRIPTION", Value: req.EvaluationMetricsSpec})
	}

	var cmd = "/scripts/run.sh"

	cpuCount := v1resource.NewMilliQuantity(int64(logCollectorMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(logCollectorMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	logCollectorContainer := v1core.Container{
		Name:    logCollectorContainerName,
		Image:   logCollectorImage,
		Command: []string{"bash", "-c", cmd},
		Env:     vars,
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
		},
		VolumeMounts:    []v1core.VolumeMount{jobVolumeMount},
		ImagePullPolicy: defaultPullPolicy,
	}
	logr.Debugf("logCollectorContainer name: %s\n", logCollectorContainer.Name)
	return logCollectorContainer
}

func constructLoadTrainingDataContainer(jobVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {
	mountPath := PodLevelJobDir

	// Construct the environment variables to pass to the container.
	// Include all the variables in the job that start with "DATA_STORE_"
	vars := make([]v1core.EnvVar, 0, len(jobEnvVars))
	prefix := "DATA_STORE_"
	for _, ev := range jobEnvVars {
		if strings.HasPrefix(ev.Name, prefix) {
			if ev.Name == "DATA_STORE_APIKEY" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_PASSWORD", Value: ev.Value})
			} else if ev.Name == "DATA_STORE_OBJECTID" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_BUCKET", Value: ev.Value})
			} else {
				vars = append(vars, ev)
			}
		}
		if ev.Name == "DATA_DIR" { // special case
			dataDir := path.Join(mountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	cpuCount := v1resource.NewMilliQuantity(int64(loadTrainingDataMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(loadTrainingDataMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	command := fmt.Sprintf(`load.sh |tee -a %s/load-data.log`, PodLevelLogDir)
	cmd := wrapCommand(command, loadDataContainerName, PodLevelJobDir)
	container := v1core.Container{
		Name:    loadDataContainerName,
		Image:   dataBrokerImageName(vars),
		Command: []string{"sh", "-c", cmd},
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
		},
		VolumeMounts:    []v1core.VolumeMount{jobVolumeMount},
		Env:             vars,
		ImagePullPolicy: defaultPullPolicy,
	}
	return container
}

func constructLoadModelContainer(jobVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {
	mountPath := PodLevelJobDir

	// Construct the environment variables to pass to the container.
	// Include all the variables in the job that start with "MODEL_STORE_"
	var vars []v1core.EnvVar
	vars = append(vars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAME", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	vars = append(vars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAMESPACE", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.namespace"}}})

	prefix := "MODEL_STORE_"
	for _, ev := range jobEnvVars {
		if strings.HasPrefix(ev.Name, prefix) {
			name := strings.Replace(ev.Name, "MODEL_STORE_", "DATA_STORE_", 1)
			if name == "DATA_STORE_APIKEY" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_PASSWORD", Value: ev.Value})
			} else if name == "DATA_STORE_OBJECTID" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_OBJECT", Value: ev.Value})
			} else {
				vars = append(vars, v1core.EnvVar{Name: name, Value: ev.Value})
			}
		}
		if ev.Name == "MODEL_DIR" { // special case
			dataDir := path.Join(mountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	command := fmt.Sprintf(`loadmodel.sh |tee -a %s/load-model.log`, PodLevelLogDir)
	cmd := wrapCommand(command, loadModelContainerName, PodLevelJobDir)

	cpuCount := v1resource.NewMilliQuantity(int64(loadModelMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(loadModelMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	container := v1core.Container{
		Name:    loadModelContainerName,
		Image:   dataBrokerImageName(vars),
		Command: []string{"sh", "-c", cmd},
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
		},
		VolumeMounts:    []v1core.VolumeMount{jobVolumeMount},
		Env:             vars,
		ImagePullPolicy: defaultPullPolicy,
	}
	return container
}

func constructLearnerContainer(req *service.JobDeploymentRequest, learnerID int,
	learnerTag string, envVars []v1core.EnvVar, volumeMount v1core.VolumeMount, logr *logger.LocLoggingEntry, mountTrainingDataStoreInLearner, mountResultsStoreInLearner, mountSSHCertsInLearner bool) v1core.Container {

	mountPath := PodLevelJobDir

	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	learnerImage := fmt.Sprintf("%s/%s_gpu_%s:%s", dockerRegistry, req.Framework, req.Version, learnerTag)

	cpuCount := v1resource.NewMilliQuantity(int64(float64(req.Resources.Cpus)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(req.Resources.Gpus), v1resource.DecimalSI)
	memInBytes := int64(calcMemory(req.Resources) * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)
	logr.Debugf("req.Resources: %+v", req.Resources)
	logr.Debugf("learner: cpu %+v, gpu %+v, mem %+v", cpuCount, gpuCount, memCount)

	volumes := []v1core.VolumeMount{
		volumeMount,
	}

	if mountSSHCertsInLearner {
		sshVolumeMount := v1core.VolumeMount{
			Name:      "sshcertmount-" + req.Name,
			MountPath: "/etc/ssh-certs",
		}
		volumes = append(volumes, sshVolumeMount)
	}

	filteredVars := getLearnerContainerEnvVars(envVars)
	vars := make([]v1core.EnvVar, 0, len(filteredVars))
	for _, ev := range filteredVars {
		if strings.HasSuffix(ev.Name, "_DIR") {
			var dir string
			if mountTrainingDataStoreInLearner && ev.Name == "DATA_DIR" {
				dir = "/mnt/" + ev.Value
				inputVolume := v1core.VolumeMount{
					Name:      "cosinputmount-" + req.Name,
					MountPath: dir,
				}
				volumes = append(volumes, inputVolume)
			} else if mountResultsStoreInLearner && ev.Name == "RESULT_DIR" {
				dir = "/mnt/" + ev.Value
				outputVolume := v1core.VolumeMount{
					Name:      "cosoutputmount-" + req.Name,
					MountPath: dir,
				}
				volumes = append(volumes, outputVolume)
			} else {
				// Adjust the paths to be in the mount point.
				dir = path.Join(mountPath, ev.Value)
			}
			vars = append(vars, v1core.EnvVar{Name: ev.Name, Value: dir})
		} else {
			vars = append(vars, ev)
		}
	}

	vars = append(vars, v1core.EnvVar{Name: "JOB_STATE_DIR", Value: PodLevelJobDir})

	command := "for i in ${!ALERTMANAGER*} ${!DLAAS*} ${!ETCD*} ${!GRAFANA*} ${!HOSTNAME*} ${!KUBERNETES*} ${!MONGO*} ${!PUSHGATEWAY*}; do unset $i; done;"
	if mountResultsStoreInLearner {
		// this commented out version writes stderr and stdout to different files
		//command = fmt.Sprintf(`%s bash -c 'train.sh > >(tee -a $RESULT_DIR/$TRAINING_ID.stdout %s/latest-log) 2> >(tee -a $RESULT_DIR/$TRAINING_ID.stderr >&2); exit ${PIPESTATUS[0]}'`, command, PodLevelJobDir)
		command = fmt.Sprintf(`%s export RESULT_DIR=$RESULT_DIR/$TRAINING_ID && mkdir -p $RESULT_DIR/learner-$LEARNER_ID; bash -c 'train.sh 2>&1 | tee -a $RESULT_DIR/learner-$LEARNER_ID/training-logs.txt %s/latest-log; exit ${PIPESTATUS[0]}'`, command, PodLevelJobDir)
	} else {
		command = fmt.Sprintf(`%s mkdir -p $RESULT_DIR; bash -c 'train.sh 2>&1 | tee -a %s/latest-log; exit ${PIPESTATUS[0]}'`, command, PodLevelJobDir)
	}
	cmd := wrapCommand(command, learnerContainerName, PodLevelJobDir)

	// Set the resourceGPU to "nvidia.com/gpu" if you want to run your GPU workloads using device plugin.
	var resourceGPU v1core.ResourceName = v1core.ResourceNvidiaGPU

	learnerContainer := v1core.Container{
		Name:            learnerContainerName,
		Image:           learnerImage,
		ImagePullPolicy: defaultPullPolicy,
		Command:         []string{"bash", "-c", cmd},
		Env:             vars,
		Ports: []v1core.ContainerPort{
			v1core.ContainerPort{ContainerPort: int32(workerPort), Protocol: v1core.ProtocolTCP},
			v1core.ContainerPort{ContainerPort: int32(sshPort), Protocol: v1core.ProtocolTCP},
		},
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:       *cpuCount,
				v1core.ResourceMemory:    *memCount,
				resourceGPU: *gpuCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:       *cpuCount,
				v1core.ResourceMemory:    *memCount,
				resourceGPU: *gpuCount,
			},
		},
		VolumeMounts: volumes,
	}
	logr.Debugf("learnerContainer name: %s\n", learnerContainer.Name)
	extendLearnerContainer(&learnerContainer, req)
	return learnerContainer
}

// Given a set of environment variables, return the subset that should appear in the learner container.
func getLearnerContainerEnvVars(allVars []v1core.EnvVar) []v1core.EnvVar {
	vars := make([]v1core.EnvVar, 0, 0)
	for _, ev := range allVars {
		if _, exists := learnerContainerEnvVars[ev.Name]; exists {
			vars = append(vars, ev)
		} else {
			// don't include this var.
		}
	}
	return vars
}

func constructStoreLogsContainer(jobVolumeMount v1core.VolumeMount, learnerID int, jobEnvVars []v1core.EnvVar) v1core.Container {
	command := "store.sh"
	container := constructStoreContainer(storeLogsContainerName, command, jobVolumeMount, learnerID, jobEnvVars)

	for i := range container.Env {
		if container.Env[i].Name == "DATA_STORE_BUCKET" {
			value := fmt.Sprintf("%s/learner-%d", container.Env[i].Value, learnerID) // per-learner directory
			container.Env[i].Value = value
		} else if container.Env[i].Name == "DATA_DIR" {
			value := fmt.Sprintf("%s/logs", PodLevelJobDir)
			container.Env[i].Value = value
		}
	}

	return container
}

func constructStoreResultsContainer(jobVolumeMount v1core.VolumeMount, learnerID int, jobEnvVars []v1core.EnvVar) v1core.Container {
	command := "true" // a no-op
	if learnerID == 1 {
		command = "store.sh" // only store results from first learner
	}
	container := constructStoreContainer(storeResultsContainerName, command, jobVolumeMount, learnerID, jobEnvVars)
	return container
}

func constructStoreContainer(containerName string, command string, jobVolumeMount v1core.VolumeMount, learnerID int, jobEnvVars []v1core.EnvVar) v1core.Container {
	mountPath := PodLevelJobDir

	// Construct the environment variables to pass to the container.
	// Include all the variables in the job that start with "DATA_STORE_"
	var vars []v1core.EnvVar
	vars = append(vars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAME", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	vars = append(vars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAMESPACE", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.namespace"}}})

	prefix := "RESULT_STORE_"
	for _, ev := range jobEnvVars {
		if strings.HasPrefix(ev.Name, prefix) {
			name := strings.Replace(ev.Name, "RESULT_STORE_", "DATA_STORE_", 1)
			if name == "DATA_STORE_APIKEY" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_PASSWORD", Value: ev.Value})
			} else if name == "DATA_STORE_OBJECTID" {
				vars = append(vars, v1core.EnvVar{Name: "DATA_STORE_BUCKET", Value: ev.Value})
			} else {
				vars = append(vars, v1core.EnvVar{Name: name, Value: ev.Value})
			}
		}
		if ev.Name == "RESULT_DIR" { // special case
			dataDir := path.Join(mountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	cpuCount := v1resource.NewMilliQuantity(int64(storeResultsMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(storeResultsMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	cmd := wrapCommand(command, containerName, PodLevelJobDir)
	container := v1core.Container{
		Name:    containerName,
		Image:   dataBrokerImageName(vars),
		Command: []string{"sh", "-c", cmd},
		Resources: v1core.ResourceRequirements{
			Requests: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
			Limits: v1core.ResourceList{
				v1core.ResourceCPU:    *cpuCount,
				v1core.ResourceMemory: *memCount,
			},
		},
		VolumeMounts:    []v1core.VolumeMount{jobVolumeMount},
		Env:             vars,
		ImagePullPolicy: defaultPullPolicy,
	}
	return container
}

// Wrap a single command with start and exit files.
func wrapCommand(cmd string, containerName string, controlFilesDirectory string) string {

	vars := map[string]string{
		"Name": containerName,
		"Dir":  controlFilesDirectory,
		"Cmd":  cmd,
	}

	var buf bytes.Buffer
	tmpl, _ := template.New("wrapped command").Parse(`
		# Don't repeat if already executed.
		if [ -f {{.Dir}}/{{.Name}}.exit ]; then
			while true; do sleep 1000; done
		fi
		# Wait for start signal.
		while [ ! -f {{.Dir}}/{{.Name}}.start ]; do sleep 2; done
		{{.Cmd}} # do the actual work
		echo $? > {{.Dir}}/{{.Name}}.exit
		while true; do sleep 2; done
	`)
	tmpl.Execute(&buf, vars)

	return buf.String()
}

func constructVolumeClaim(name string, namespace string, volumeSize int64, labels map[string]string) *v1core.PersistentVolumeClaim {
	claim, err := GetVolumeClaim(volumeSize)
	if err != nil {
		return nil
	}
	claim.Name = name
	claim.Namespace = namespace
	claim.Labels = labels
	claim.Spec.AccessModes = []v1core.PersistentVolumeAccessMode{v1core.ReadWriteMany}
	return claim
}

//LearnerDataDir ... Return the training data directory.
func LearnerDataDir(envVars []v1core.EnvVar) string {
	return getValue(envVars, "DATA_DIR")
}

// Return the value of the named environment variable.
func getValue(envVars []v1core.EnvVar, name string) string {
	value := ""
	for _, ev := range envVars {
		if ev.Name == name {
			value = ev.Value
			break
		}
	}
	return value
}

// Return the Docker image of the data broker for this set of variables
func dataBrokerImageName(vars []v1core.EnvVar) string {
	t := defaultDatabrokerType
	for _, ev := range vars {
		if ev.Name == "DATA_STORE_TYPE" {
			if ev.Value == "s3_datastore" {
				t = "s3"
				break
			}
		} else if ev.Name == "DATA_STORE_TYPE" {
			// t should be a string like "s3" or "objectstorage", but we expect to also receive
			// strings like "s3_datastore" (read in from the .ini files), hence strip the suffix here.
			storeType := strings.Replace(ev.Value, "_datastore", "", 1)
			if contains(validDatabrokerTypes, storeType) {
				t = storeType
				break
			}
		}
	}
	//servicesTag := viper.GetString(config.ServicesTagKey)
	//logr.Debugf("servicesTag: %s", servicesTag)
	//TODO: Tag the databroker and statusrecorder images
	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	dataBrokerTag := viper.GetString(config.DataBrokerTagKey)
	imageName := dataBrokerImageNameExtended(dockerRegistry, t, dataBrokerTag)
	return imageName
}

// Checks whether a value is contained in an array
func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
