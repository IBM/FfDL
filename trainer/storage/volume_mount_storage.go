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

package storage

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
)

const (
	// DataStoreTypeNotImplemented is the name that represents an in-memory data store
	DataStoreTypeVolumeMount = "volume_mount_datastore"
)

type volumeMountStorage struct {
	conf    map[string]string
}

// NewVolumeMountStorage implements an in-memor object store for testing.
func NewVolumeMountStorage(conf map[string]string) (DataStore, error) {
	if conf == nil {
		return nil, fmt.Errorf("conf argument is nil")
	}
	vms := new(volumeMountStorage)
	vms.conf = conf
	return vms, nil
}

func (o *volumeMountStorage) Connect() error {
	// nothing to do
	return nil
}

func (o *volumeMountStorage) UploadArchive(container string, object string, payload []byte) error {
	return nil
}

func (o *volumeMountStorage) DownloadArchive(container string, object string) ([]byte, error) {
	return nil, fmt.Errorf("container or object '%s/%s' not found", container, object)
}

func (o *volumeMountStorage) DeleteArchive(container string, object string) error {
	log.Debugf("volumeMountStorage")
	return fmt.Errorf("DeleteArchive Not Implemented")
}

func (o *volumeMountStorage) GetTrainedModelSize(path string, numLearners int32) (int64, error) {
	return 0, fmt.Errorf("GetTrainedModelSize Not Implemented")

}

func (o *volumeMountStorage) DownloadTrainedModelAsZipStream(path string, numLearners int32, writer io.Writer) error {
	return fmt.Errorf("DownloadTrainedModelAsZipStream Not Implemented")
}

func (o *volumeMountStorage) DownloadTrainedModelLogFile(path string, numLearners int32, learnerIndex int32, objectPath string, writer io.Writer) error {
	return fmt.Errorf("DownloadTrainedModelLogFile Not Implemented")
}

func (o *volumeMountStorage) ContainerExists(name string) (bool, error) {
	return false, fmt.Errorf("ContainerExists Not Implemented")
}

func (o *volumeMountStorage) Disconnect() {
	// nothing to do
}
