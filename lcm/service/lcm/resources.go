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
	//"errors"
	"strconv"
	"time"

	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/IBM/FfDL/commons/config"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
)

func (s *lcmService) currentResourceSnapshot(jdreq *service.JobDeploymentRequest, numLearners int, logr *logger.LocLoggingEntry) bool {

	cpusRequired := float64(numLearners) * float64(jdreq.Resources.Cpus)
	gpusRequired := int64(numLearners) * int64(jdreq.Resources.Gpus)
	continueDeploy := true

	//Account for Job Monitor
	cpusRequired++

	if numLearners > 1 {
		//Account for CPU usage by the parameter server
		cpusRequired = cpusRequired + float64(jdreq.Resources.Cpus)
	}

	k8sConnected, alloc, rreq, avl := getResources(s, logr)
	if !k8sConnected {
		continueDeploy = false
		logr.Debugf("(LCM) Cannot connect to kubernetes to deploy %s", jdreq.TrainingId)
		errUpd := updateJobStatus(jdreq.TrainingId, grpc_trainer_v2.Status_FAILED, jdreq.UserId, service.StatusMessages_INTERNAL_ERROR.String(), errCodeK8SConnection, logr)
		if errUpd != nil {
			logr.Errorln("(deployDistributedTrainingJob) Before deploying job, error while calling Trainer service client update: ", errUpd.Error())
		}
		return continueDeploy
	}

	logr.Debugf("(LCM) Logging current resource usage stats before deploying Training Job %s", jdreq.TrainingId)
	logr.Debugf("(LCM) %f CPUs and %d GPUs required for Training Job %s", cpusRequired, gpusRequired, jdreq.TrainingId)
	logr.Debugf("(LCM) %f CPUs and %f GPUs allocatable", alloc.cpusAllocatable, alloc.gpusAllocatable)
	logr.Debugf("(LCM) %f CPUs and %f GPUs requested already by existing pods", rreq.cpusRequested, rreq.gpusRequested)
	logr.Debugf("(LCM) %f CPUs and %f GPUs available for Training Job %s", avl.cpusAvailable, avl.gpusAvailable, jdreq.TrainingId)
	logr.Debugf("(LCM) %f GB RAM requested already by existing pods", rreq.memRequested/(1024*1024*1024))
	logr.Debugf("(LCM) %f GB RAM available for Training Job %s", avl.memAvailable/(1024*1024*1024), jdreq.TrainingId)
	return continueDeploy

}

func (s *lcmService) resourceSnapshotOnDeletion(jkreq *service.JobKillRequest, logr *logger.LocLoggingEntry) {

	k8sConnected, alloc, rreq, avl := getResources(s, logr)

	if !k8sConnected {
		logr.Debugf("(LCM) Cannot connect to kubernetes to determine resource snapshot on deletion")
	}

	logr.Debugf("(LCM) Logging current resource usage stats after deleting Training Job %s", jkreq.TrainingId)
	logr.Debugf("(LCM) %f CPUs and %f GPUs allocatable", alloc.cpusAllocatable, alloc.gpusAllocatable)
	logr.Debugf("(LCM) %f CPUs and %f GPUs requested already by existing pods", rreq.cpusRequested, rreq.gpusRequested)
	logr.Debugf("(LCM) %f CPUs and %f GPUs available for Training Job %s", avl.cpusAvailable, avl.gpusAvailable, jkreq.TrainingId)
	logr.Debugf("(LCM) %f GB RAM requested already by existing pods", rreq.memRequested/(1024*1024*1024))
	logr.Debugf("(LCM) %f GB RAM available for Training Job %s", avl.memAvailable/(1024*1024*1024), jkreq.TrainingId)

}

type allocatableResources struct {
	cpusAllocatable float64
	gpusAllocatable float64
	memAllocatable  float64
}

type requestedResources struct {
	cpusRequested float64
	gpusRequested float64
	memRequested  float64
}

type availableResources struct {
	cpusAvailable float64
	gpusAvailable float64
	memAvailable  float64
}

func getResources(s *lcmService, logr *logger.LocLoggingEntry) (bool, *allocatableResources, *requestedResources, *availableResources) {

	var cpusAllocatable, cpusRequested, cpusAvailable float64
	cpusAllocatable, cpusRequested, cpusAvailable = 0.0, 0.0, 0.0

	var memAllocatable, memRequested, memAvailable float64
	memAllocatable, memRequested, memAvailable = 0.0, 0.0, 0.0

	var gpusAllocatable, gpusRequested, gpusAvailable float64
	gpusAllocatable, gpusRequested, gpusAvailable = 0.0, 0.0, 0.0

	//Selector to select everything in the kubernetes cluster
	//selector := labels.SelectorFromSet(labels.Set{})

	k8sConnected := true

	//Get all then nodes and then the pods
	nodes, err := s.k8sClient.Core().Nodes().List(metav1.ListOptions{})
	pods, err1 := s.k8sClient.Core().Pods(config.GetLearnerNamespace()).List(metav1.ListOptions{})

	i := 1

	//Retry if there is an error in accessing kubernetes, with 30s sleeps in between tries
	for (err != nil || err1 != nil) && i <= numRetries {
		logr.Infof("There was an error in accessing Kubernetes to determine available resources. Retrying")
		nodes, err = s.k8sClient.Core().Nodes().List(metav1.ListOptions{})
		pods, err1 = s.k8sClient.Core().Pods(config.GetLearnerNamespace()).List(metav1.ListOptions{})

		if (err != nil || err1 != nil) && i == numRetries {
			logr.Infof("Accessing kubernetes to get a snapshot of current resource usage failed. Giving up after %d retries", numRetries)
			logr.WithError(err).Errorf("Error while getting list of nodes from Kubernetes ")
			logr.WithError(err1).Errorf("Error while getting list of pods from Kubernetes ")
		}

		time.Sleep(30 * time.Second)

		i++

		if i == numRetries {
			k8sConnected = false
		}
	}

	// Set the resourceGPU to "nvidia.com/gpu" if you want to run your GPU workloads using device plugin.
	resourceGPU := v1core.ResourceNvidiaGPU

	//By querying nodes, determine the number of allocatable resources
	for _, node := range nodes.Items {
		availableResources := node.Status.Allocatable

		cpuQty := availableResources[v1core.ResourceCPU]
		cpu, _ := strconv.ParseFloat(cpuQty.AsDec().String(), 64)
		memQty := availableResources[v1core.ResourceMemory]
		mem, _ := strconv.ParseFloat(memQty.AsDec().String(), 64)
		gpuQty := availableResources[resourceGPU]
		gpu, _ := strconv.ParseFloat(gpuQty.AsDec().String(), 64)

		cpusAllocatable += cpu
		memAllocatable += mem
		gpusAllocatable += gpu

	}

	for _, pod := range pods.Items {
		logr.Debugf("Inspecting pod %s", pod.ObjectMeta.Name)
		containers := pod.Spec.Containers
		for j := 0; j < len(containers); j++ {
			resourcesRequested := containers[j].Resources.Requests
			cpuQty := resourcesRequested[v1core.ResourceCPU]
			cpu, _ := strconv.ParseFloat(cpuQty.AsDec().String(), 64)
			memQty := resourcesRequested[v1core.ResourceMemory]
			mem, _ := strconv.ParseFloat(memQty.AsDec().String(), 64)
			gpuQty := resourcesRequested[resourceGPU]
			gpu, _ := strconv.ParseFloat(gpuQty.AsDec().String(), 64)

			cpusRequested += cpu
			memRequested += mem
			gpusRequested += gpu
		}
	}

	cpusAvailable = cpusAllocatable - cpusRequested
	gpusAvailable = gpusAllocatable - gpusRequested
	memAvailable = memAllocatable - memRequested

	alloc := &allocatableResources{
		cpusAllocatable: cpusAllocatable,
		gpusAllocatable: gpusAllocatable,
		memAllocatable:  memAllocatable,
	}

	rreq := &requestedResources{
		cpusRequested: cpusRequested,
		gpusRequested: gpusRequested,
		memRequested:  memRequested,
	}

	avl := &availableResources{
		cpusAvailable: cpusAvailable,
		gpusAvailable: gpusAvailable,
		memAvailable:  memAvailable,
	}

	return k8sConnected, alloc, rreq, avl

}
