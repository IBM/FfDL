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
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/lcm/service/lcm/learner"
)

func (t nonSplitTraining) Start() error {

	gpus := make(map[string]string)
	if t.req.Resources.Gpus > 0 {
		gpus["ibm-cloud.kubernetes.io/gpu-type"] = t.req.Resources.GpuType
	}

	learnerDefn := t.learner
	helperDefn := t.helper

	helperAndLearnerVolumes := append(learnerDefn.volumes, helperDefn.etcdVolume, helperDefn.sharedVolume)
	helperContainers := t.constructAuxillaryContainers()

	//now create the learner container
	learnerContainer := constructLearnerContainer(t.req, learnerDefn.envVars, learnerDefn.volumeMounts, helperDefn.sharedVolumeMount, learnerDefn.mountTrainingDataStoreInLearner, learnerDefn.mountResultsStoreInLearner, t.logr)
	helperContainers = append(helperContainers, learnerContainer)

	//create pod, service, statefuleset spec
	nonSplitLearnerPodSpec := learner.CreatePodSpec(helperContainers, helperAndLearnerVolumes, map[string]string{"training_id": t.req.TrainingId, "user_id": t.req.UserId}, gpus)
	serviceSpec := learner.CreateServiceSpec(learnerDefn.name, t.req.TrainingId)
	statefulSetSpec := learner.CreateStatefulSetSpecForLearner(learnerDefn.name, serviceSpec.Name, learnerDefn.numberOfLearners, nonSplitLearnerPodSpec)

	numLearners := int(t.req.GetResources().Learners)

	return t.CreateFromBOM(&nonSplitTrainingBOM{
		learnerDefn.secrets,
		serviceSpec,
		statefulSetSpec,
		numLearners,
	})

}

//CreateFromBOM ... eventually use with controller and make this transactional
func (t nonSplitTraining) CreateFromBOM(bom *nonSplitTrainingBOM) error {
	logr := t.logr
	namespace := config.GetLearnerNamespace()

	for _, secret := range bom.secrets {
		//create the secrets
		if _, err := t.k8sClient.CoreV1().Secrets(namespace).Create(secret); err != nil {
			logr.WithError(err).Errorf("Failed in creating secrets %s while deploying for training ", secret.Name)
			return err
		}
	}

	if bom.numLearners > 1 {
		//create service
		if _, err := t.k8sClient.CoreV1().Services(namespace).Create(bom.service); err != nil {
			logr.WithError(err).Errorf("Failed in creating service %s while deploying for training ", bom.service.Name)
			return err
		}
	}

	//create the stateful set
	if _, err := t.k8sClient.AppsV1beta1().StatefulSets(namespace).Create(bom.learnerBOM); err != nil {
		logr.WithError(err).Errorf("Failed in creating statefulsets %s while deploying for training ", bom.learnerBOM.Name)
		return err
	}

	return nil

}
