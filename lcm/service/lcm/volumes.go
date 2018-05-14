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
	"errors"

	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
)

// from https://github.ibm.com/alchemy-containers/armada-storage-file-plugin/blob/master/armada-storage-classes
var supportedVolumeSizes = []v1resource.Quantity{
	v1resource.MustParse("20Gi"),
	v1resource.MustParse("20Gi"),
	v1resource.MustParse("40Gi"),
	v1resource.MustParse("80Gi"),
	v1resource.MustParse("100Gi"),
	v1resource.MustParse("250Gi"),
	v1resource.MustParse("500Gi"),
	v1resource.MustParse("1Ti"),
	v1resource.MustParse("2Ti"),
	v1resource.MustParse("4Ti"),
}

// GetVolumeClaim returns a PersistentVolumeClaim struct for the given volume size (specified in bytes).
func GetVolumeClaim(volumeSize int64, logr *logger.LocLoggingEntry) (*v1core.PersistentVolumeClaim, error) {
	quantity := getStorageQuantity(volumeSize)
	if quantity == nil {
		err := errors.New("Unable to find matching storage quantity")
		logr.WithError(err).Debugf("getStorageQuantity returned error")
		return nil, err
	}

	class := getStorageClass(volumeSize)
	if class == "" {
		err := errors.New("Unable to find matching storage class")
		logr.WithError(err).Debugf("getStorageClass returned error")
		return nil, err
	}

	claim := &v1core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"volume.beta.kubernetes.io/storage-class": class,
			},
		},
		Spec: v1core.PersistentVolumeClaimSpec{
			Resources: v1core.ResourceRequirements{
				Requests: v1core.ResourceList{
					v1core.ResourceStorage: *quantity,
				},
			},
		},
	}
	return claim, nil
}

// Return the storage class for the given volume size.
func getStorageClass(volumeSize int64) string {
	return viper.GetString(config.SharedVolumeStorageClassKey)
}

// Return the storage quantity for the given volume size.
// This will round up to the nearest available quantity
func getStorageQuantity(volumeSize int64) *v1resource.Quantity {
	for _, q := range supportedVolumeSizes {
		if q.CmpInt64(volumeSize) >= 0 {
			// Found a match
			return &q
		}
	}
	return nil
}
