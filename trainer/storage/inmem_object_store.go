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
	"archive/zip"
	"fmt"
	"io"
	"strings"
	"sync"
	"strconv"

	log "github.com/sirupsen/logrus"
)

const (
	// DataStoreTypeInMemory is the name that represents an in-memory data store
	DataStoreTypeInMemory = "inmem_datastore"
)

var (
	// store needs to be global as we different instances to access the same data
	store                     map[string]map[string][]byte
	createOnce sync.Once
)

type inMemObjectStore struct {
}

// NewInMemObjectStore implements an in-memor object store for testing.
func NewInMemObjectStore(conf map[string]string) (DataStore, error) {
	createOnce.Do(func() {
		store = make(map[string]map[string][]byte)
	})
	return &inMemObjectStore{}, nil
}

func (o *inMemObjectStore) Connect() error {
	// nothing to do
	return nil
}

func (o *inMemObjectStore) UploadArchive(container string, object string, payload []byte) error {
	log.Infof("(fake) UploadArchive: container '%s', object '%s'", container, object)
	if store[container] == nil {
		store[container] = make(map[string][]byte)
	}
	store[container][object] = payload
	return nil
}

func (o *inMemObjectStore) DownloadArchive(container string, object string) ([]byte, error) {
	if store[container] != nil {
		return store[container][object], nil
	}
	return nil, fmt.Errorf("container or object '%s/%s' not found", container, object)
}

func (o *inMemObjectStore) DeleteArchive(container string, object string) error {
	if store[container] != nil {
		delete(store[container], object)

		// delete container if last item
		if len(store[container]) == 0 {
			delete(store, container)
		}
	}
	return nil
}

func (o *inMemObjectStore) GetTrainedModelSize(path string, numLearners int32) (int64, error) {
	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}

	var trainedModelSize int64
	// find the objects with objectPrefix
	objects := make(map[string][]byte)
	for k, v := range store[container] {
		if strings.HasPrefix(k, objectPrefix) {
			objects[k] = v
			size, err := strconv.Atoi(string(v))
			if err != nil {
				return 0, err
			}
			trainedModelSize += int64(size)
		}
	}
	return trainedModelSize, nil

}


func (o *inMemObjectStore) DownloadTrainedModelAsZipStream(path string, numLearners int32, writer io.Writer) error {
	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}

	// find the objects with objectPrefix
	objects := make(map[string][]byte)
	for k, v := range store[container] {
		if strings.HasPrefix(k, objectPrefix) {
			objects[k] = v
		}
	}

	log.Debugf("Trained model object names: %v", objects)

	w := zip.NewWriter(writer)

	for k, v := range objects {

		f, err := w.Create(k)
		if err != nil {
			return err
		}
		_, err = f.Write(v)
		if err != nil {
			return err
		}
	}

	err := w.Close()
	return err
}

func (o *inMemObjectStore) DownloadTrainedModelLogFile(path string, numLearners int32, learnerIndex int32, objectPath string, writer io.Writer) error {
	// separate path in container and object
	log.Debugf("inMemObjectStore DownloadTrainedModelLogFile: bucket: %s, key: %s", path, objectPath)
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}

	// find the objects with objectPrefix
	objects := make(map[string][]byte)
	for k, v := range store[container] {
		if strings.HasPrefix(k, objectPrefix) {
			objects[k] = v
		}
	}

	log.Debugf("Trained model object names: %v", objects)

	fullObjectPath := objectPrefix + "/" + objectPath

	foundit := false
	for k, v := range objects {
		if fullObjectPath == k {
			_, err := writer.Write(v)
			if err != nil {
				return err
			}
			foundit = true
			break
		}
	}
	if foundit == false {
		return fmt.Errorf("DownloadTrainedModelLogFile: Could not find object: %s", fullObjectPath)
	}

	return nil
}

func (o *inMemObjectStore) ContainerExists(name string) (bool, error) {
	if _, ok := store[name]; ok {
		return true, nil
	}
	return false, nil
}

func (o *inMemObjectStore) Disconnect() {
	// nothing to do
}
