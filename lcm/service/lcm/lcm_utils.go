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
	"io/ioutil"
	"math/rand"
	"strconv"
	"time"

	"github.com/IBM/FfDL/commons/config"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/commons/service"
	"github.com/IBM/FfDL/commons/util"

	"github.com/IBM/FfDL/trainer/client"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"
	"golang.org/x/net/context"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

//creates all the znodes used by a training job before it is deployed
func createEtcdNodes(lcm *lcmService, jobName string, userID string, trainingID string, numOfLearners int, framework string, logr *logger.LocLoggingEntry) error {
	pathToValueMapping := map[string]string{
		trainingID + "/" + zkNotes:                             "",
		trainingID + "/" + zkUserID:                            userID,
		trainingID + "/" + zkFramework:                         framework,
		trainingID + "/" + zkLearners + "/" + zkTotLearners:    string(numOfLearners),
		trainingID + "/" + zkJobName:                           jobName,
		trainingID + "/" + zkLearners + "/" + zkLearnerLock:    "",
		trainingID + "/" + zkLearners + "/" + zkLearnerCounter: "1",
		trainingID + "/" + zkLearners + "/" + zkAliveLearners:  "0",
		trainingID + "/" + zkGlobalCursor + "/" + zkGCState:    "0",
	}

	for path, val := range pathToValueMapping {
		pathCreated, error := lcm.etcdClient.PutIfKeyMissing(path, val, logr)
		if error != nil {
			return error
		}
		if !pathCreated {
			return fmt.Errorf("Failed to create the path %v , since it was already present", path)
		}
	}

	if numOfLearners > 1 {
		path := trainingID + "/" + "parameter-server"
		pathCreated, error := lcm.etcdClient.PutIfKeyMissing(path, "", logr)
		if error != nil {
			return error
		}
		if !pathCreated {
			return fmt.Errorf("Failed to create the path %v , since it was already present", path)
		}
	}

	return nil
}

//helper function to construct a job monitor name from job name
func constructJMName(jobName string) string {
	jmName := "jobmonitor-" + jobName
	return jmName
}

//helper function to construct a learner name from job name
func constructLearnerName(learnerID int, jobName string) string {
	return "learner-" + strconv.Itoa(learnerID) + "-" + jobName
}

//helper function to construct a learnerHelper name from job name
func constructLearnerHelperName(learnerID int, jobName string) string {
	return "lhelper-" + strconv.Itoa(learnerID) + "-" + jobName
}

//helper function to construct a learner service name from job name
func constructLearnerServiceName(learnerID int, jobName string) string {
	return constructLearnerName(learnerID, jobName)
}

//helper function to construct a learner service name from job name
func constructLearnerVolumeClaimName(learnerID int, jobName string) string {
	return constructLearnerName(learnerID, jobName)
}

//helper function to construct a parameter server name from job name
func constructPSName(jobName string) string {
	psName := "grpc-ps-" + jobName
	return psName
}

// Get the disk size (in bytes) requested for a job.
func getStorageSize(r *service.ResourceRequirements) int64 {
	// The default size for all jobs
	size := config.GetVolumeSize()

	// Use the requested volume size if it's specified
	if r.Storage > 0 {
		storageSizeInBytes := int64(calcStorage(r) * 1024 * 1024)
		size = storageSizeInBytes
	}

	return size
}

// Return the name of a volume to use for a job.
func getStaticVolume(logr *logger.LocLoggingEntry) string {

	// Read file with PVC specs.
	logr.Debugf("entry into getStaticVolume")
	pvcsFile := "/etc/static-volumes/PVCs.yaml"
	bytes, err := ioutil.ReadFile(pvcsFile)
	if err != nil {
		logr.Warnf("Unable to load %s: %s", pvcsFile, err)
		return ""
	}

	// Read top level object.
	obj, err := runtime.Decode(scheme.Codecs.UniversalDeserializer(), bytes)
	if err != nil {
		logr.Errorf("Error decoding volume spec file: %s", err)
		return ""
	}

	// Make sure it's a list.
	var pvcList *v1core.List

	switch obj := obj.(type) {
	case *v1core.List:
		pvcList = obj
	//case *v1core.PersistentVolumeClaim:
	//	pvcList = new(v1core.List)
	//	pvcList = append(pvcList.Items, obj)
	default:
		logr.Errorf("Decoded object is not a list.")
		return ""
	}

	// Iterate through list.
	volumes := []string{}
	for _, item := range pvcList.Items {
		// Parse for PVC object
		pvcObj, err := runtime.Decode(scheme.Codecs.UniversalDeserializer(), item.Raw)
		if err != nil {
			logr.Errorf("Error decoding item in list: %s", err)
			continue
		}
		var pvc *v1core.PersistentVolumeClaim
		switch pvcObj := pvcObj.(type) {
		case *v1core.PersistentVolumeClaim:
			pvc = pvcObj
		default:
			logr.Errorf("Decoded object is not a PVC.")
			continue
		}

		// Add PVC name to list.
		volumes = append(volumes, pvc.Name)
	}

	// Return a random volume.
	logr.Debugf("static volumes: %s", volumes)
	if len(volumes) > 0 {
		n := rand.Int() % len(volumes)
		logr.Debugf("returning a volume: %d", n)
		return volumes[n]
	}

	return ""
}

func handleDeploymentFailure(s *lcmService, dlaasJobName string, tID string,
	userID string, component string, logr *logger.LocLoggingEntry) {

	logr.Errorf("updating status to FAILED")
	if errUpd := updateJobStatus(tID, grpc_trainer_v2.Status_FAILED, userID, service.StatusMessages_INTERNAL_ERROR.String(), client.ErrCodeFailedDeploy, logr); errUpd != nil {
		logr.WithError(errUpd).Errorf("after failed %s, error while calling Trainer service client update", component)
	}

	//Cleaning up resources out of an abundance of caution
	logr.Errorf("training FAILED so going ahead and cleaning up resources")
	if errKill := s.killDeployedJob(dlaasJobName, tID, userID); errKill != nil {
		logr.WithError(errKill).Errorf("after failed %s, problem calling KillDeployedJob for job ", component)
	}

}

func jobBasePath(trainingID string) string {
	return config.GetEtcdPrefix() + trainingID
}

// Return the etcd base path of learner znodes.
func learnerEtcdBasePath(trainingID string) string {
	return jobBasePath(trainingID) + "/learners"
}

// Return the etcd base path of status of learner znodes.
func learnerNodeEtcdStatusPath(trainingID string, learnerID int) string {
	return fmt.Sprintf("%s/learner_%d/status", learnerEtcdBasePath(trainingID), learnerID)
}

func learnerNodeEtcdStatusPathRelative(trainingID string, learnerID int) string {
	return fmt.Sprintf("%s/learner_%d/status", trainingID, learnerID)
}

// Return the etcd base path of learner znodes.
func learnerNodeEtcdBasePath(trainingID string, learnerID int) string {
	return fmt.Sprintf("%s/learner_%d/", learnerEtcdBasePath(trainingID), learnerID)
}

// calcMemory is a utility to convert the memory from DLaaS resource requirements
// to the default MB notation
func calcMemory(r *service.ResourceRequirements) float64 {
	return calcSize(r.Memory, r.MemoryUnit)
}

// calcStorage is a utility to convert the storage from DLaaS resource requirements
// to the default MB notation
func calcStorage(r *service.ResourceRequirements) float64 {
	return calcSize(r.Storage, r.StorageUnit)
}

// calcSize converts from memory resource requirements to the default MB notation
func calcSize(size float64, unit service.ResourceRequirements_MemoryUnit) float64 {
	// according to google unit converter :)
	switch unit {
	case service.ResourceRequirements_MiB:
		return util.RoundPlus(size*1.048576, 2)
	case service.ResourceRequirements_GB:
		return util.RoundPlus(size*1000, 2)
	case service.ResourceRequirements_TB:
		return util.RoundPlus(size*1000*1000, 2)
	case service.ResourceRequirements_GiB:
		return util.RoundPlus(size*1073.741824, 2)
	case service.ResourceRequirements_TiB:
		return util.RoundPlus(size*1073.741824*1073.741824, 2)
	default:
		return size // assume MB
	}
}

//update job status in the database
//update job status in cassandra
func updateJobStatus(trainingID string, updStatus grpc_trainer_v2.Status, userID string, statusMessage string, errorCode string, logr *logger.LocLoggingEntry) error {
	logr.Debugf("(updateJobStatus) Updating status of %s to %s", trainingID, updStatus.String())
	updateRequest := &grpc_trainer_v2.UpdateRequest{TrainingId: trainingID, Status: updStatus, UserId: userID, StatusMessage: statusMessage, ErrorCode: errorCode}

	trainer, err := client.NewTrainer()
	if err != nil {
		logr.WithError(err).Errorf("(updateJobStatus) Creating training client for status update failed. Training ID %s New Status %s", trainingID, updStatus.String())
		logr.Errorf("(updateJobStatus) Error while creating training client is %s", err.Error())
	}
	defer trainer.Close()
	err = util.Retry(10, 100*time.Millisecond, "UpdateTrainingJob", logr, func() error {
		//ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
		//defer cancel()
		_, err = trainer.Client().UpdateTrainingJob(context.Background(), updateRequest)
		if err != nil {
			logr.WithError(err).Error("Failed to update status to the trainer. Retrying")
			logr.Infof("WARNING: Status updates for %s may be temporarily inconsistent due to failure to communicate with Trainer.", trainingID)
		}
		return err
	})
	if err != nil {
		logr.WithError(err).Errorf("Failed to update status to the trainer. Already retried several times.")
		logr.Infof("WARNING : Status of job %s will likely be incorrect", trainingID)
		return err
	}

	logr.Debugf("(updateJobStatus) Status update request for %s sent to trainer", trainingID)
	return nil
}

func isJobDone(jobStatus string, logr *logger.LocLoggingEntry) bool {
	statusUpdate := client.GetStatus(jobStatus, logr)
	status := statusUpdate.Status
	return status == grpc_trainer_v2.Status_COMPLETED || status == grpc_trainer_v2.Status_FAILED || status == grpc_trainer_v2.Status_HALTED
}

// Set the DLaaS service type label to an object.
// This label is used to configure Calico network policy rules for the pod.
func setServiceTypeLabel(spec *metav1.ObjectMeta, value string) {
	spec.Labels["service"] = value
}
