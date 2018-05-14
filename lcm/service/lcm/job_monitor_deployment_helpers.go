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
	"strconv"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/lcm/lcmconfig"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"

	"github.com/spf13/viper"
	v1beta1 "k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

//Populate all the environment variables used to deploy learner jobs on Kubernetes
func populateJobMonitorEnvVariablesAndLabels(req *service.JobDeploymentRequest, trainingID string, jobName string, userID string, numLearners int, useNativeDistribution bool) ([]v1core.EnvVar, map[string]string) {

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

	envVars := []v1core.EnvVar{
		v1core.EnvVar{
			Name:  "USE_NATIVE_DISTRIBUTION",
			Value: strconv.FormatBool(useNativeDistribution),
		},
		v1core.EnvVar{
			Name:  "TRAINING_ID",
			Value: trainingID,
		},
		v1core.EnvVar{
			Name:  "JOB_NAME",
			Value: jobName,
		},
		v1core.EnvVar{
			Name:  "USER_ID",
			Value: userID,
		},
		v1core.EnvVar{
			Name:  "NUM_LEARNERS",
			Value: strconv.Itoa(numLearners),
		},
		v1core.EnvVar{
			Name:  "DLAAS_PUSH_METRICS_ENABLED",
			Value: strconv.FormatBool(true),
		},

		getEnvVarFromLCMSecret("DLAAS_ETCD_ADDRESS"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_USERNAME"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PASSWORD"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PREFIX"),
		v1core.EnvVar{
			Name:  "DLAAS_ENV",
			Value: config.GetValue(config.EnvKey),
		},
		v1core.EnvVar{
			Name:  "DLAAS_LOGLEVEL",
			Value: config.GetValue(config.LogLevelKey),
		},
		v1core.EnvVar{
			Name:  "DLAAS_POD_NAMESPACE",
			Value: config.GetPodNamespace(),
		},
		v1core.EnvVar{
			Name:  "DLAAS_LEARNER_KUBE_NAMESPACE",
			Value: config.GetLearnerNamespace(),
		},
	}

	// add all labels passed from the user API
	jobLabels := make(map[string]string)
	for k, v := range req.Labels {
		jobLabels[k] = v
	}

	return envVars, jobLabels
}

func defineJobMonitorDeployment(req *service.JobDeploymentRequest, envVars []v1core.EnvVar, jmLabels map[string]string, logr *logger.LocLoggingEntry) *v1beta1.Deployment {

	jmTag := viper.GetString(config.DLaaSImageTagKey)

	dockerRegistry := ""

	//Decide where to get job monitor image from by looking at DLAAS_ENV. That is pointed to by config.EnvKey
	//registry.ng.bluemix.net/* is not accessible from minikube on laptops
	if viper.GetString(config.LCMDeploymentKey) == config.HybridEnv {
		dockerRegistry = viper.GetString(config.IBMDockerRegistryKey)
	} else {
		dockerRegistry = viper.GetString(config.LearnerRegistryKey)
	}

	jmImage := jobmonitorImageNameExtended(dockerRegistry, jmTag)
	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)
	logr.Debugf("jmImage: %s, imagePullSecret: %s, imagePullPolicy: %s",
		jmImage, imagePullSecret, lcmconfig.GetImagePullPolicy())

	cpuCount := v1resource.NewMilliQuantity(int64(float64(0.5)*1000.0), v1resource.DecimalSI)
	memInBytes := int64(512 * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)
	logr.Debugf("job monitor: cpu %+v, mem %+v", cpuCount, memCount)

	jmName := constructJMName(req.Name)

	deploySpec := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: jmName,
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
					Name: jmName,
					Labels: map[string]string{
						"app":         jmName,
						"training_id": req.TrainingId,
						"service":     "dlaas-jobmonitor",
						"user_id":     req.UserId,
					},
				},
				Spec: v1core.PodSpec{
					Volumes: []v1core.Volume{
						v1core.Volume{
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
						},
					},
					Containers: []v1core.Container{
						v1core.Container{
							Name:  jmName,
							Image: jmImage,
							//Command: [],
							Env: envVars,
							VolumeMounts: []v1core.VolumeMount{
								v1core.VolumeMount{
									Name:      "etcd-ssl-cert",
									MountPath: "/etc/certs/",
									ReadOnly:  true,
								},
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
							ImagePullPolicy: lcmconfig.GetImagePullPolicy(),
						},
					},
					RestartPolicy: v1core.RestartPolicyAlways,
					DNSPolicy:     v1core.DNSClusterFirst,
					ImagePullSecrets: []v1core.LocalObjectReference{
						v1core.LocalObjectReference{
							Name: imagePullSecret,
						},
					},
				},
			},
		},
	}

	logr.Debug("defineJobMonitorDeployment() Pull Secret: %v", imagePullSecret)

	return deploySpec
}
