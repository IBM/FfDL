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
	"time"

	"github.com/IBM/FfDL/commons/config"

	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/commons/util"

	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/IBM/FfDL/commons/logger"

	v1beta1 "k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func populatePSEnvVariablesAndLabels(req *service.JobDeploymentRequest, logr *logger.LocLoggingEntry) []v1core.EnvVar {
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

	numLearners := int(req.GetResources().Learners)

	additionalEnvVars := []v1core.EnvVar{
		v1core.EnvVar{
			Name:  "JOBID",
			Value: "1111",
		},
		v1core.EnvVar{
			Name:  "NUM_LEARNERS",
			Value: strconv.Itoa(numLearners),
		},
		v1core.EnvVar{
			Name:  "TCP_PORT",
			Value: strconv.Itoa(int(psPort)),
		},
		v1core.EnvVar{
			Name:  "ZK_DIR",
			Value: req.TrainingId + "/parameter-server",
		},
		v1core.EnvVar{
			Name:  "ZK_DIR",
			Value: req.TrainingId + "/parameter-server",
		},
		getEnvVarFromLCMSecret("DLAAS_ETCD_ADDRESS"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_USERNAME"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PASSWORD"),
		getEnvVarFromLCMSecret("DLAAS_ETCD_PREFIX"),
		v1core.EnvVar{
			Name:  "FOR_TEST",
			Value: "1",
		},
		v1core.EnvVar{
			Name:  "DLAAS_JOB_ID",
			Value: req.TrainingId,
		},
		v1core.EnvVar{
			Name:  "ZNODE_NAME",
			Value: "singleshard",
		},
	}

	envVars := additionalEnvVars[:]

	for k, v := range req.EnvVars {
		item := v1core.EnvVar{
			Name:  k,
			Value: v,
		}
		envVars = append(envVars, item)
	}
	return envVars

}

func definePSDeployment(req *service.JobDeploymentRequest, envVars []v1core.EnvVar, logr *logger.LocLoggingEntry) *v1beta1.Deployment {

	learnerTag := viper.GetString(config.LearnerTagKey)
	logr.Debugf("deployParameterServer (LCM) learnerTag: %s", learnerTag)
	dockerRegistry := viper.GetString(config.LearnerRegistryKey)
	psImage := fmt.Sprintf("%s/parameter-server:%s", dockerRegistry, learnerTag)

	cpuCount := v1resource.NewMilliQuantity(int64(float64(req.Resources.Cpus)*1000.0), v1resource.DecimalSI)
	memInBytes := int64(calcMemory(req.Resources) * 1024 * 1024)
	memCount := v1resource.NewQuantity(memInBytes, v1resource.DecimalSI)
	logr.Debugf("(LCM definePSDeployment) Parameter server: cpu %+v, mem %+v", cpuCount, memCount)

	psName := constructPSName(req.Name)

	deploySpec := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: psName,
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
					Name: psName,
					Labels: map[string]string{
						"app":         psName,
						"training_id": req.TrainingId,
						"service":     "dlaas-parameter-server",
					},
				},
				Spec: v1core.PodSpec{
					Containers: []v1core.Container{
						v1core.Container{
							Name:  psName,
							Image: psImage,
							//Command: []string{"/usr/bin/supervisord"},
							Env: envVars,
							Ports: []v1core.ContainerPort{
								v1core.ContainerPort{ContainerPort: int32(psPort), Protocol: v1core.ProtocolTCP},
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
								v1core.VolumeMount{
									Name:      "etcd-ssl-cert-vol",
									MountPath: "/etc/certs/",
									ReadOnly:  true,
								},
							},
							ImagePullPolicy: v1core.PullAlways,
						},
					},
					Volumes: []v1core.Volume{v1core.Volume{
						Name: "etcd-ssl-cert-vol",
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
					}},
					RestartPolicy: v1core.RestartPolicyAlways,
					DNSPolicy:     v1core.DNSClusterFirst,
					ImagePullSecrets: []v1core.LocalObjectReference{
						v1core.LocalObjectReference{
							Name: viper.GetString(config.LearnerImagePullSecretKey),
						},
					},
				},
			},
		},
	}

	return deploySpec

}

func definePSService(psName string, trainingID string) *v1core.Service {
	// Define service spec.
	serviceSpec := &v1core.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: psName,
			Labels: map[string]string{
				"app":         psName,
				"training_id": trainingID,
				"service":     "dlaas-parameter-server",
			},
		},
		Spec: v1core.ServiceSpec{
			Type:     v1core.ServiceTypeClusterIP,
			Selector: map[string]string{"app": psName},
			Ports: []v1core.ServicePort{
				v1core.ServicePort{
					Name:     "grpc",
					Protocol: v1core.ProtocolTCP,
					Port:     psPort,
				},
			},
		},
	}

	return serviceSpec
}

func deployParameterServer(ctx context.Context, s *lcmService, req *service.JobDeploymentRequest) error {
	logr := logger.LocLogger(InitLogger(req.TrainingId, req.UserId))
	psName := constructPSName(req.Name)

	envVars := populatePSEnvVariablesAndLabels(req, logr)

	deploySpec := definePSDeployment(req, envVars, logr)

	err := util.Retry(10, 10*time.Second, "CreateParameterServerDeployment", logr, func() error {
		psDeploy, err := s.k8sClient.AppsV1beta1().Deployments(config.GetLearnerNamespace()).Create(deploySpec)
		if err != nil {
			logr.WithError(err).Errorf("(LCM deployParameterServer) Retrying after failure to create parameter server deployment: %s\n", deploySpec)
			return err
		}
		logr.Debugf("Parameter Server Deployment is %s\n", psDeploy)
		return nil
	})

	if err != nil {
		logr.WithError(err).Errorf("Failed to create parameter server deployment: %s\n", deploySpec)
		logr.Errorf("********************************************************")
		return err
	}

	serviceSpec := definePSService(psName, req.TrainingId)

	err = util.Retry(10, 10*time.Second, "CreateParameterServerService", logr, func() error {
		psSvc, err := s.k8sClient.Core().Services(config.GetLearnerNamespace()).Create(serviceSpec)
		if err != nil {
			logr.WithError(err).Errorf("(LCM deployParameterServer) Retrying after failure to create parameter server service: %s\n", serviceSpec)
			return err
		}
		logr.Debugf("Parameter Server Service is %s\n", psSvc)
		return nil
	})

	if err != nil {
		logr.WithError(err).Errorf("After trying several times, Failed to create parameter server service: %s\n", serviceSpec)
		logr.Errorf("********************************************************")
		return err
	}

	logr.Infof("Finished Creating Parameter Server\n")

	return nil

}
