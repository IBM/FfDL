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

/*
func TestCreateLearnerFromBOMForSingleLearnerNonSplitModeNonCOS(t *testing.T) {

	mountTrainingDataStoreInLearner, mountResultsStoreInLearner, mountSSHCertsInLearner := false, false, false
	trainingID := "TestCreateLearnerFromBOMForSingleLearnerNonSplitModeNonCOS"
	//create pod, service, statefuleset spec
	nonSplitLearnerPodSpec := CreatePodSpec("podname", trainingID, helperContainers, learnerVolumes)
	serviceSpec := CreateServiceSpec("service", trainingID)
	statefulSetSpec := CreateStatefulSetSpecForLearner("statefulset", serviceSpec.Name, 1, nonSplitLearnerPodSpec)

	//now wrap it with stateful set
	dependencies := StatefulSetDependencies{
		Secrets: createSecretsForTest(),
		Volumes: helperAndLearnerVolumes,
		Service: serviceSpec,
	}

	CreateLearnerFromBOM(statefulSetSpec, dependencies)

}

func createSecretsForTest() []*v1core.Secret {
	mockSecrets := Secrets{
		ResultsDirSecret:   &COSVolumeSecret{},
		TrainingDataSecret: &COSVolumeSecret{},
		SSHVolumeSecret:    &SSHVolumeSecret{},
	}
	return CreateVolumeSecretsSpec(mockSecrets)

}
*/
