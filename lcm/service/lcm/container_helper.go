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

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/lcm/lcmconfig"
	"github.com/IBM/FfDL/lcm/service/lcm/learner"
	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/viper"

	"bytes"

	"github.com/IBM/FfDL/commons/logger"
	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const logCollectorContainerName string = "log-collector" // the name of the learner container in the pod
const loadDataContainerName = "load-data"
const loadModelContainerName = "load-model"
const learnerContainerName = "learner"
const storeResultsContainerName = "store-results"
const storeLogsContainerName = "store-logs"
const learnerConfigDir = "/etc/learner-config"

const simpleLogCollectorName = "log_collector"

const logCollectorBadTagNoTDSFound = "dummy-tag-no-tds-found"

// valid names of databroker types that map to "databroker_<type>" Docker image names
var validDatabrokerTypes = []string{"objectstorage", "s3"}

// default databroker type
var defaultDatabrokerType = "objectstorage"

const (
	workerPort int32 = 2222
	sshPort    int32 = 22
)

//helpers won't know about learner ID since that is only available to stateful set learners
// TODO used by controller (for tracking state of individual learners) and logs, do we really need this for logs
// need to use 1 and not 0 because job monitor tracks path starting with learner 1 and not 0
const masterLearnerID = 1

func constructControllerContainer(trainingID string, etcdVolumeMount, sharedVolumeMount v1core.VolumeMount, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) v1core.Container {

	learnerNodeBasePath := learnerNodeEtcdBasePath(trainingID, masterLearnerID)
	learnerNodeStatusPath := learnerNodeEtcdStatusPath(trainingID, masterLearnerID)
	jobBasePath := jobBasePath(trainingID)

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

	servicesTag := viper.GetString(config.ServicesTagKey)

	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	controllerImageName := controllerImageNameExtended(dockerRegistry, servicesTag)

	cmd := fmt.Sprintf("controller.sh")

	// short-circuit the load and store databrokers when we mount object storage directly
	if mountResultsStoreInLearner {
		cmd = "echo 0 > " + sharedVolumeMount.MountPath + "/store-results.exit && " + cmd
	}
	if mountTrainingDataStoreInLearner {
		cmd = "echo 0 > " + sharedVolumeMount.MountPath + "/load-data.exit && " + cmd
	}

	cpuCount := v1resource.NewMilliQuantity(int64(controllerMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(controllerMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	container := v1core.Container{
		Name:    "controller",
		Image:   controllerImageName,
		Command: []string{"sh", "-c", cmd},
		Env: []v1core.EnvVar{
			v1core.EnvVar{Name: "JOB_STATE_DIR", Value: sharedVolumeMount.MountPath},
			v1core.EnvVar{Name: "JOB_LEARNER_ZNODE_PATH", Value: learnerNodeBasePath},
			v1core.EnvVar{Name: "JOB_BASE_PATH", Value: jobBasePath},
			v1core.EnvVar{Name: "JOB_LEARNER_ZNODE_STATUS_PATH", Value: learnerNodeStatusPath},
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
		VolumeMounts:    []v1core.VolumeMount{etcdVolumeMount, sharedVolumeMount},
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
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
					// For the moment we're just going to use TF 1.3, but the tag should change to be non-version
					// specific, and we should just use latest TF.
					logCollectorImageShortName = fmt.Sprintf("%s_extract", "tensorboard")
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

func constructLogCollector(sharedVolumeMount v1core.VolumeMount, k8sClient kubernetes.Interface, req *service.JobDeploymentRequest,
	envVars []v1core.EnvVar, logr *logger.LocLoggingEntry) v1core.Container {

	defaultTag := findTrainingDataServiceTag(k8sClient, logr)

	logCollectorImageShortName, learnerEMTag := fetchImageNameFromEvaluationMetrics(req.EvaluationMetricsSpec, defaultTag, req.Framework, req.Version, logr)

	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	logCollectorImage :=
		fmt.Sprintf("%s/%s:%s", dockerRegistry, logCollectorImageShortName, learnerEMTag)

	vars := make([]v1core.EnvVar, 0, len(envVars))
	for _, ev := range envVars {
		if strings.HasSuffix(ev.Name, "_DIR") {
			// Adjust the paths to be in the mount point.
			dir := path.Join(sharedVolumeMount.MountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: ev.Name, Value: dir})
		} else {
			vars = append(vars, ev)
		}
	}

	vars = append(vars, v1core.EnvVar{Name: "JOB_STATE_DIR", Value: sharedVolumeMount.MountPath})
	vars = append(vars, v1core.EnvVar{Name: "TRAINING_DATA_NAMESPACE", Value: config.GetPodNamespace()})

	if req.EvaluationMetricsSpec != "" {
		vars = append(vars, v1core.EnvVar{Name: "EM_DESCRIPTION", Value: req.EvaluationMetricsSpec})
	}

	cpuCount := v1resource.NewMilliQuantity(int64(logCollectorMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(logCollectorMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	logCollectorContainer := v1core.Container{
		Name:    logCollectorContainerName,
		Image:   logCollectorImage,
		Command: []string{"bash", "-c", "/scripts/run.sh"},
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
		VolumeMounts:    []v1core.VolumeMount{sharedVolumeMount},
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
	}
	return logCollectorContainer
}

func constructLoadTrainingDataContainer(sharedVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {

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
			dataDir := path.Join(sharedVolumeMount.MountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	cpuCount := v1resource.NewMilliQuantity(int64(loadTrainingDataMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(loadTrainingDataMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	command := fmt.Sprintf(`load.sh |tee -a %s/load-data.log`, PodLevelLogDir)
	cmd := wrapCommand(command, loadDataContainerName, sharedVolumeMount.MountPath, false)
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
		VolumeMounts:    []v1core.VolumeMount{sharedVolumeMount},
		Env:             vars,
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
	}
	return container
}

func constructLoadModelContainer(sharedVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {

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
			dataDir := path.Join(sharedVolumeMount.MountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	command := "loadmodel.sh"
	cmd := wrapCommand(command, loadModelContainerName, sharedVolumeMount.MountPath, false)

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
		VolumeMounts:    []v1core.VolumeMount{sharedVolumeMount},
		Env:             vars,
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
	}
	return container
}

func constructLearnerContainer(req *service.JobDeploymentRequest, envVars []v1core.EnvVar, learnerVolumeMounts []v1core.VolumeMount, sharedVolumeMount v1core.VolumeMount, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool, logr *logger.LocLoggingEntry) v1core.Container {
	cpuCount := v1resource.NewMilliQuantity(int64(float64(req.Resources.Cpus)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(req.Resources.Gpus), v1resource.DecimalSI)
	memInBytes := int64(calcMemory(req.Resources) * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	//argh!!! this should be abstracted out as well
	command := "for i in ${!ALERTMANAGER*} ${!DLAAS*} ${!ETCD*} ${!GRAFANA*} ${!HOSTNAME*} ${!KUBERNETES*} ${!MONGO*} ${!PUSHGATEWAY*}; do unset $i; done;"
	//FIXME need to have the learner IDs start from 1 rather than 0
	var cmd string
	var doCondExitWrite = true
	if mountResultsStoreInLearner {
		//this $RESULT_DIR/training-log.txt is a top level log, where logs of train.sh get redirected
		command = fmt.Sprintf(`
		function copyLogs() {
             if [ ! -f "%[2]s/%[4]s.exit" ]; then
				 echo Wait for start-logs start signal.
				 # Wait for start-logs start signal.
				 start_wait=$(date +%%s)
				 while [ ! -f %[2]s/%[4]s.start ]; do
					end_wait=$(date +%%s) ;
					duration_wait=$((end_wait-start_wait)) ;
					# Make this a pretty long timeout, as we really want the lc to finish
					if [ ${duration_wait} -gt 120 ]; then
						echo "time out waiting for %[2]s/%[4]s.start"
						break
					fi
					sleep 2;
				 done ;
				 echo Calling copy logs.
				 mv -f $LOG_DIR/* $RESULT_DIR/learner-$LEARNER_ID ;
				 ERROR_CODE=$? ;
				 echo $ERROR_CODE > $RESULT_DIR/learner-$LEARNER_ID/.log-copy-complete ;
				 echo "Setting status ${ERROR_CODE} > %[2]s/%[4]s.exit"
				 echo $ERROR_CODE > %[2]s/%[4]s.exit ;
			 fi
		}
		
		%[1]s export LEARNER_ID=$((${DOWNWARD_API_POD_NAME##*-} + 1)) ;
		 export RESULT_DIR=$RESULT_DIR/$TRAINING_ID ;
		 mkdir -p $RESULT_DIR/learner-$LEARNER_ID ;
		 mkdir -p $CHECKPOINT_DIR ;
         # Leaving this commmented code as a reminder that trap handling may be part of the answer
		 # trap "copyLogs" SIGHUP SIGINT SIGTERM
		 bash -c ' train.sh 2>&1 |  tee -a $RESULT_DIR/learner-$LEARNER_ID/training-log.txt %[2]s/latest-log ; exit ${PIPESTATUS[0]}'
		 main_cmd_status=$?
		 echo learner "done, snooze before exit (temporary, to be removed)"
         echo "Setting status ${main_cmd_status} > %[2]s/%[3]s.exit"
		 echo $main_cmd_status > %[2]s/%[3]s.exit
         echo "call copyLogs function"
         copyLogs
         echo "learner container exiting"
         `,
			command, sharedVolumeMount.MountPath, learnerContainerName,
			storeLogsContainerName)
	} else {
		command = fmt.Sprintf(`%s mkdir -p $RESULT_DIR ; bash -c ' train.sh 2>&1 | tee -a %s/latest-log; exit ${PIPESTATUS[0]}'`, command, sharedVolumeMount.MountPath)
		doCondExitWrite = false
	}
	cmd = wrapCommand(command, learnerContainerName, sharedVolumeMount.MountPath, doCondExitWrite)

	container := learner.Container{
		Image: learner.Image{Framework: req.Framework, Version: req.Version, Tag: req.EnvVars["DLAAS_LEARNER_IMAGE_TAG"]},
		Resources: learner.Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: append(learnerVolumeMounts, sharedVolumeMount),
		Name:         learnerContainerName,
		EnvVars:      envVars,
		Command:      cmd,
	}

	learnerContainer := learner.CreateContainerSpec(container)
	extendLearnerContainer(&learnerContainer, req, logr)
	//logr.Debugf("learnerContainer: %v+", learnerContainer)
	// dumpAsYaml("learnerContainer", learnerContainer, logr)

	return learnerContainer
}

func constructStoreLogsContainer(sharedVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {

	command := "store.sh"
	container := constructStoreContainer(storeLogsContainerName, command, sharedVolumeMount, jobEnvVars)

	for i := range container.Env {
		if container.Env[i].Name == "DATA_STORE_BUCKET" {
			value := fmt.Sprintf("%s/learner-%d", container.Env[i].Value, masterLearnerID) // per-learner directory
			container.Env[i].Value = value
		} else if container.Env[i].Name == "DATA_DIR" {
			value := fmt.Sprintf("%s/logs", sharedVolumeMount.MountPath)
			container.Env[i].Value = value
		}
	}

	return container
}

func constructStoreResultsContainer(sharedVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {

	//FIXME how does this work in terms of split learner
	command := "store.sh" // only store results from first learner
	container := constructStoreContainer(storeResultsContainerName, command, sharedVolumeMount, jobEnvVars)
	return container
}

func constructStoreContainer(containerName, command string, sharedVolumeMount v1core.VolumeMount, jobEnvVars []v1core.EnvVar) v1core.Container {

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
			dataDir := path.Join(sharedVolumeMount.MountPath, ev.Value)
			vars = append(vars, v1core.EnvVar{Name: "DATA_DIR", Value: dataDir})
		}
	}

	cpuCount := v1resource.NewMilliQuantity(int64(storeResultsMilliCPU), v1resource.DecimalSI)
	memInBytes := int64(storeResultsMemInMB * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)

	cmd := wrapCommand(command, containerName, sharedVolumeMount.MountPath, false)
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
		VolumeMounts:    []v1core.VolumeMount{sharedVolumeMount},
		Env:             vars,
		ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
	}
	return container
}

// Wrap a single command with start and exit files.
func wrapCommand(cmd string, containerName string, controlFilesDirectory string, doCondExitWrite bool) string {

	vars := map[string]string{
		"Name": containerName,
		"Dir":  controlFilesDirectory,
		"Cmd":  cmd,
	}

	var exitWriteStr string
	if doCondExitWrite {
		exitWriteStr = `
		if [ ! -f {{.Dir}}/{{.Name}}.exit ]; then
			echo $main_cmd_status > {{.Dir}}/{{.Name}}.exit
        fi
		`
	} else {
		exitWriteStr = `
		echo $? > {{.Dir}}/{{.Name}}.exit
		`
	}

	var buf bytes.Buffer
	tmpl, _ := template.New("wrapped command").Parse(`
		# Don't repeat if already executed.
		if [ -f {{.Dir}}/{{.Name}}.exit ]; then
			while true; do sleep 1000; done
		fi
		# Wait for start signal.
		while [ ! -f {{.Dir}}/{{.Name}}.start ]; do sleep 2; done
		# Record the start time. Note: In distributed mode, this
		# file will get overwritten by each learner (this is intentional)
		date "+%s%N" | cut -b1-13 > {{.Dir}}/{{.Name}}.start_time
		{{.Cmd}} # do the actual work` + exitWriteStr +
		`while true; do sleep 2; done
	`)
	tmpl.Execute(&buf, vars)

	return buf.String()
}

func constructVolumeClaim(name string, namespace string, volumeSize int64,
	labels map[string]string, logr *logger.LocLoggingEntry) *v1core.PersistentVolumeClaim {
	claim, err := GetVolumeClaim(volumeSize, logr)
	if err != nil {
		logr.WithError(err).Debugf("GetVolumeClaim returned error")
		return nil
	}
	claim.Name = name
	claim.Namespace = namespace
	claim.Labels = labels
	claim.Spec.AccessModes = []v1core.PersistentVolumeAccessMode{v1core.ReadWriteMany}
	return claim
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
