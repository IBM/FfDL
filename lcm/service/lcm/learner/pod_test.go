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

package learner

import (
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
)

func createPodSpecForTesting() v1core.PodTemplateSpec {
	prefix := "nonSplitSingleLearner-"
	learnerContainer := createNonSplitSinglerLearnerContainer()
	volumes := []v1core.Volume{} //no volumes since non split

	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)

	return CreatePodSpec([]v1core.Container{learnerContainer}, volumes, map[string]string{"training_id": prefix + "trainingID"}, map[string]string{}, imagePullSecret)

}

func createNonSplitSinglerLearnerContainer() v1core.Container {

	//Create only learner container since there is no good way to mock the other containers for now
	var envars []v1core.EnvVar
	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)

	container := Container{
		Image: Image{Framework: "tensorflow", Version: "1.5", Tag: "latest"},
		Resources: Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: []v1core.VolumeMount{},
		Name:         "test-learner-container",
		EnvVars:      envars,
		Command:      "echo hello",
	}

	return CreateContainerSpec(container)
}
