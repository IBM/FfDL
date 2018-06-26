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
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/IBM/FfDL/commons/logger"
)

const (
	// intDevBMObjectStoreURL is the internal development env SL proxy to the Bluemix ObjectStore v3
	intDevBMObjectStoreURL = "https://proxy-gateway-d.watsonplatform.net/softlayer/identity/v3"
	// untStagingBMObjectStoreURL is the internal staging env SL proxy to the Bluemix ObjectStore v3
	intStagingBMObjectStoreURL = "https://proxy-gateway-s.watsonplatform.net/softlayer/identity/v3"
	// intProdBMObjectStoreURL is the internal production env SL proxy to the Bluemix ObjectStore v3
	intProdBMObjectStoreURL = "https://identity.open.softlayer.com/v3"
	// extBMObjectStoreURL is the external SL url to the Bluemix ObjectStore v3
	extBMObjectStoreURL = "https://identity.open.softlayer.com/v3"
	// intSLObjectStoreURLFragment is the interanal SL Object store URL fragment
	intSLObjectStoreURLFragment = "objectstorage.service.networklayer.com"
	// extSLObjectStoreURLFragment is the external SL Object store URL fragment
	extSLObjectStoreURLFragment = "objectstorage.softlayer.net"

	// UsernameKey is the key for the user name
	UsernameKey = "user_name"
	// PasswordKey is the key for the password
	PasswordKey = "password"
	// AuthURLKey is the key for the authentication URL
	AuthURLKey  = "auth_url"
	// DomainKey is the key for the domain name
	DomainKey   = "domain_name"
	// RegionKey is the key for the region name
	RegionKey   = "region"
	// ProjectKey is the key for the project ID
	ProjectKey  = "project_id"
	// StorageType is the key for the storage type
	StorageType = "type"
)

// ErrNotConnected is thrown when the ObjectStore is not connected (i.e., authenticated)
var ErrNotConnected = errors.New("not connected to object store")

// DataStore is a minimal interface for interacting with data stores such as IBM ObjectStore or other backend
// for uploading and downloading DL models, logs, etc.
type DataStore interface {
	Connect() error
	Disconnect()
	ContainerExists(name string) (bool, error)
	UploadArchive(container string, object string, payload []byte) error
	DownloadArchive(container string, object string) ([]byte, error)
	DeleteArchive(container string, object string) error
	GetTrainedModelSize(path string, numLearners int32) (int64, error)
	DownloadTrainedModelAsZipStream(path string, numLearners int32, writer io.Writer) error
	DownloadTrainedModelLogFile(path string, numLearners int32, learnerIndex int32, filename string, writer io.Writer) error
}

//
// Helper functions useful accross object store implementations
//
func retry(attempts int, interval time.Duration, description string, logr *logger.LocLoggingEntry, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(interval)
		logr.Warnf("Retrying function %s", description)
	}
	return fmt.Errorf("function %s after %d attempts, last error: %s", description, attempts, err)
}
