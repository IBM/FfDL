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

package helper

import (
	"k8s.io/client-go/kubernetes"

	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
)

//ETCDVolume ...
type ETCDVolume struct {
	Name      string
	MountSpec VolumeMountSpec
}

//LocalVolume ...
type LocalVolume struct {
	Name      string
	MountSpec VolumeMountSpec
}

//SharedNFSVolume ...
type SharedNFSVolume struct {
	Name, PVCClaimName string
	PVC                *v1core.PersistentVolumeClaim //nil for static volumes as this is already created
	MountSpec          VolumeMountSpec
}

//Volumes ...
type Volumes struct {
	ETCDVolume                        *ETCDVolume
	SharedSplitLearnerHelperVolume    *SharedNFSVolume
	SharedNonSplitLearnerHelperVolume *LocalVolume
}

//VolumeMountSpec ...
type VolumeMountSpec struct {
	MountPath, SubPath string
}

//CreatePVCFromBOM ...
func CreatePVCFromBOM(sharedVolumeClaim *v1core.PersistentVolumeClaim, k8sClient kubernetes.Interface) error {
	namespace := config.GetLearnerNamespace()

	_, err := k8sClient.Core().PersistentVolumeClaims(namespace).Create(sharedVolumeClaim)
	return err

}

//CreateETCDVolume ...
func (volumes Volumes) CreateETCDVolume() v1core.Volume {
	return createETCDVolume(volumes.ETCDVolume.Name)
}

//CreateETCDVolumeMount ...
func (volumes Volumes) CreateETCDVolumeMount() v1core.VolumeMount {
	return createETCDVolumeMount(volumes.ETCDVolume.Name)
}

//CreateDataVolume ...
func (volumes Volumes) CreateDataVolume() v1core.Volume {

	if volumes.SharedNonSplitLearnerHelperVolume != nil {
		//local volume is required since operating in non split mode
		return localEmptyDirVolume(volumes.SharedNonSplitLearnerHelperVolume.Name)
	}

	//shared NFS volume is required
	return sharedVolume(volumes.SharedSplitLearnerHelperVolume.Name, volumes.SharedSplitLearnerHelperVolume.PVCClaimName)
}

//CreateDataVolumeMount ...
func (volumes Volumes) CreateDataVolumeMount() v1core.VolumeMount {
	if volumes.SharedNonSplitLearnerHelperVolume != nil {
		return localEmptyDirVolumeMount(volumes.SharedNonSplitLearnerHelperVolume.Name, volumes.SharedNonSplitLearnerHelperVolume.MountSpec.MountPath, volumes.SharedNonSplitLearnerHelperVolume.MountSpec.SubPath)
	}
	return sharedVolumeMount(volumes.SharedSplitLearnerHelperVolume.Name, volumes.SharedSplitLearnerHelperVolume.MountSpec.MountPath, volumes.SharedSplitLearnerHelperVolume.MountSpec.SubPath)
}

//DynamicPVCReference ...
func (volumes Volumes) DynamicPVCReference() *v1core.PersistentVolumeClaim {
	if volumes.SharedSplitLearnerHelperVolume == nil {
		return nil
	}
	return volumes.SharedSplitLearnerHelperVolume.PVC

}

func createETCDVolumeMount(name string) v1core.VolumeMount {
	return v1core.VolumeMount{
		Name:      name,
		MountPath: "/etc/certs/",
		ReadOnly:  true,
	}
}

func createETCDVolume(name string) v1core.Volume {
	// Volume with etcd certificates.
	return v1core.Volume{
		Name: name,
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
	}
}

func localEmptyDirVolume(name string) v1core.Volume {
	return v1core.Volume{
		Name:         name,
		VolumeSource: v1core.VolumeSource{EmptyDir: &v1core.EmptyDirVolumeSource{}},
	}
}

func localEmptyDirVolumeMount(name, baseDirectory, trainingID string) v1core.VolumeMount {

	return v1core.VolumeMount{
		Name:      name,
		MountPath: baseDirectory,
		SubPath:   trainingID,
	}
}

func sharedVolume(name, pvcClaimName string) v1core.Volume {
	return v1core.Volume{
		Name: name,
		VolumeSource: v1core.VolumeSource{
			PersistentVolumeClaim: &v1core.PersistentVolumeClaimVolumeSource{ClaimName: pvcClaimName},
		},
	}
}

func sharedVolumeMount(name, baseDirectory, trainingID string) v1core.VolumeMount {
	return localEmptyDirVolumeMount(name, baseDirectory, trainingID)
}
