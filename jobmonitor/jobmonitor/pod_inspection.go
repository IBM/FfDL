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

package jobmonitor

import (
	"time"

	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"

	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	trainerClient "github.com/IBM/FfDL/trainer/client"
)

func (jm *JobMonitor) checkIfJobStarted(logr *logger.LocLoggingEntry) {
	selector := "training_id==" + jm.TrainingID
	logr.Debugf("(Job Monitor checkIfJobStarted) Checking if there are kubernetes learner PODS associated with training job %s", jm.TrainingID)

	for i := 1; i <= insuffResourcesRetries; i++ {
		pods, err := jm.k8sClient.Core().Pods(config.GetLearnerNamespace()).List(metav1.ListOptions{LabelSelector: selector})

		numPending := 0
		numRunning := 0
		numFailed := 0

		numPodsExpected := jm.NumLearners + 2 //1 helper plus 1 job monitor

		if err == nil {
			for _, pod := range pods.Items {
				switch pod.Status.Phase {
				case v1core.PodRunning:
					numRunning++
					continue
				case v1core.PodPending:
					logr.Debugf("(Job Monitor checkIfJobStarted) Job %s seems to have a pending pod %s", jm.TrainingID, pod.ObjectMeta.Name)
					logr.Debugf("(Job Monitor checkIfJobStarted) Pod status message is %s Reason is %s", pod.Status.Message, pod.Status.Reason)

					conditions := pod.Status.Conditions
					for _, condition := range conditions {
						if condition.Type == v1core.PodScheduled && condition.Status == v1core.ConditionFalse {
							logr.Debugf("Pending Pod Condition reason %s message %s", condition.Reason, condition.Message)
							numPending++
						}
					}

					containerStatuses := pod.Status.ContainerStatuses
					for _, containerStatus := range containerStatuses {
						if containerStatus.State.Waiting != nil {
							reason := containerStatus.State.Waiting.Reason
							message := containerStatus.State.Waiting.Message
							logr.Debugf("Container Waiting Reason is %s message is %s", reason, message)
						}
						if containerStatus.State.Terminated != nil {
							reason := containerStatus.State.Terminated.Reason
							message := containerStatus.State.Terminated.Message
							logr.Debugf("Container Waiting Reason is %s message is %s", reason, message)
						}
					}

				case v1core.PodFailed:
					logr.Debugf("(Job Monitor checkIfJobStarted) Job %s seems to have a failed pod %s", jm.TrainingID, pod.ObjectMeta.Name)
					logr.Debugf("(Job Monitor checkIfJobStarted) Pod status message is %s Reason is %s", pod.Status.Message, pod.Status.Reason)
					numFailed++

					containerStatuses := pod.Status.ContainerStatuses
					for _, containerStatus := range containerStatuses {
						if containerStatus.State.Waiting != nil {
							reason := containerStatus.State.Waiting.Reason
							message := containerStatus.State.Waiting.Message
							logr.Debugf("Container Waiting Reason is %s message is %s", reason, message)
						}
						if containerStatus.State.Terminated != nil {
							reason := containerStatus.State.Terminated.Reason
							message := containerStatus.State.Terminated.Message
							logr.Debugf("Container Waiting Reason is %s message is %s", reason, message)
						}
					}

				}
			}
		}

		if numRunning >= numPodsExpected {
			logr.Debugf("All learner pods, one helper and one job monitor seem to have started")
			return
		}

		if i == insuffResourcesRetries && numPending >= 1 {
			jm.metrics.insufficientK8sResourcesErrorCounter.Add(1)
			updateJobStatusOnError(jm.TrainingID, jm.UserID, trainerClient.ErrCodeInsufficientResources, service.StatusMessages_INSUFFICIENT_RESOURCES.String(), logr)
			time.Sleep(30 * time.Second)
			KillDeployedJob(jm.TrainingID, jm.UserID, jm.JobName, logr)
			return
		}

		if numFailed >= 1 && i == insuffResourcesRetries {
			updateJobStatusOnError(jm.TrainingID, jm.UserID, trainerClient.ErrFailedPodReasonUnknown, service.StatusMessages_INTERNAL_ERROR.String(), logr)
			KillDeployedJob(jm.TrainingID, jm.UserID, jm.JobName, logr)
		}

		time.Sleep(30 * time.Second)
	}

}
