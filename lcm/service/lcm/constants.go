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

const (
	zkState            = "state"
	zkNotes            = "notes"
	zkJobName          = "jobname"
	zkLearners         = "learners"
	zkParamServer      = "paramservers"
	zkTotLearners      = "total_learners"
	zkLock             = "lock"
	zkShardes          = "total_shards"
	zkLcm              = "lcm"
	zkDoneLearner      = "done_learner_"
	zkLearner          = "learner_"
	zkStatus           = "status"
	zkOwner            = "owner"
	zkAliveLearners    = "alive_learners"
	zkFinishedLearners = "finished_learners"
	zkLearnerCounter   = "counter"
	zkLearnerLock      = "lock"
	zkUserID           = "userid"
	zkNodeExists       = "zk: node already exists"
	zkGlobalCursor     = "globalcursor"
	zkGCState          = "gcstate"
	zkFramework        = "framework"
)

const (
	internalInit     string = "Init"
	internalPS       string = "PS"
	internalLearners string = "Learners"
	internalRunning  string = "Running"
	internalDone     string = "Done"
)

const (
	psPort                       int32  = 50051
	caffeFrameworkName           string = "caffe"
	tfFrameworkName              string = "tensorflow"
	torchFrameworkName           string = "torch"
	caffe2FrameworkName          string = "caffe2"
	pytorchFrameworkName         string = "pytorch"
	customFrameworkName          string = "custom"
	numRetries                          = 5
	maxGPUsPerNode                      = 4

	// Not sure if these should stay or go, -sb 3/15/2018
	errCodeNormal                       = "000"
	errCodeInsufficientResources        = "100"
	errCodeFailedDeploy                 = "101"
	errCodeFailedPS                     = "102" // TODO: unused?
	errCodeImagePull                    = "103"
	errFailedPodReasonUnknown           = "104"
	errCodeK8SConnection                = "200"
	errCodeEtcdConnection               = "201"
)

const (
	//Total CPU for helpers = 2.5
	//Total RAM for helpers = 4 GB
	storeResultsMilliCPU=20
	storeResultsMemInMB=100
	loadModelMilliCPU=20
	loadModelMemInMB=50
	loadTrainingDataMilliCPU=20
	loadTrainingDataMemInMB=100
	logCollectorMilliCPU=20
	logCollectorMemInMB=100
	controllerMilliCPU=20
	controllerMemInMB=100
)

const (
	reason                          = "reason"
	framework                       = "framework"
	progress                        = "progress"
	outcome                         = "outcome"
	halted                          = "job_halted"
	started                         = "job_started"
	jmLaunchFailed                  = "jm_launch_failed"
	psLaunchFailed                  = "ps_launch_failed"
	learnerLaunchFailed             = "learner_launch_failed"
	killed                          = "job_killed"
	servicesDeletedPhaseComplete    = "servicesDeletedPhaseComplete"
	deploymentsDeletedPhaseComplete = "deploymentsDeletedPhaseComplete"
	replicaSetsDeletedPhaseComplete = "replicaSetsDeletedPhaseComplete"
	jobsDeletedPhaseComplete        = "jobsDeletedPhaseComplete"
	podsDeletedPhaseComplete        = "podsDeletedPhaseComplete"
	pvsDeletedPhaseComplete         = "pvsDeletedPhaseComplete"
	secretsDeletedPhaseComplete     = "secretsDeletedPhaseComplete"
	etcdKeysDeletedPhaseComplete    = "etcdKeysDeletedPhaseComplete"
)
