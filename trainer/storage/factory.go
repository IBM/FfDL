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
	"strings"

	log "github.com/sirupsen/logrus"
)

// DataStoreFactory is a simple factory for creating a specific
type DataStoreFactory func(conf map[string]string) (DataStore, error)

var datastoreFactories = make(map[string]DataStoreFactory)

func init() {
	Register(DataStoreTypeMountVolume, NewVolumeMountStorage)
	Register(DataStoreTypeMountCOSS3, NewS3ObjectStore)
	Register(DataStoreTypeS3, NewS3ObjectStore)
	Register(DataStoreTypeSwift, NewSwiftObjectStore)
	Register(DataStoreTypeInMemory, NewInMemObjectStore)

	// TODO remove this after we reworked all the examples.
	Register("softlayer_objectstore", NewSwiftObjectStore)
	Register("bluemix_objectstore", NewSwiftObjectStore)
}

// Register enables registration of available data stores.
func Register(name string, factory DataStoreFactory) {
	if factory == nil {
		log.Panicf("Datastore factory %s does not exist.", name)
	}
	_, registered := datastoreFactories[name]
	if registered {
		log.Errorf("Datastore factory %s already registered. Ignoring.", name)
	}
	datastoreFactories[name] = factory
}

// CreateDataStore is a factory for instantiating the according to the configuration.
func CreateDataStore(dataStoreName string, conf map[string]string) (DataStore, error) {
	engineFactory, ok := datastoreFactories[dataStoreName]
	if !ok {
		// Factory has not been registered.
		// Make a list of all available datastore factories for logging.
		availableDatastores := make([]string, 2)
		for k := range datastoreFactories {
			availableDatastores = append(availableDatastores, k)
		}
		return nil, fmt.Errorf(fmt.Sprintf("Invalid data store name: %s. Must be one of: %s", dataStoreName, strings.Join(availableDatastores, ", ")))
	}
	return engineFactory(conf) // instantiate the factory
}
