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
	"errors"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/FfDL/commons/config"

	log "github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/logger"
	"github.com/ncw/swift"
	"github.com/spf13/viper"
)

const (
	// DataStoreTypeSwift is the type string for the Swift-based object store.
	DataStoreTypeSwift = "swift_datastore"
)

type swiftObjectStore struct {
	conf map[string]string
	conn *swift.Connection
}

// NewSwiftObjectStore creates a new connector for the object store. If `conn` is set to nil,
// the configuration is read via viper. If conn is provided, it will use it for connecting
// to the provided object store.
func NewSwiftObjectStore(conf map[string]string) (DataStore, error) {
	if conf == nil {
		return nil, fmt.Errorf("conf is nil")
	}

	// check config and fatal if not found
	config.FatalOnAbsentKeyInMap(AuthURLKey, conf)
	config.FatalOnAbsentKeyInMap(UsernameKey, conf)
	config.FatalOnAbsentKeyInMap(PasswordKey, conf)
	// region and domain are optional

	return &swiftObjectStore{
		conf: conf,
	}, nil
}

func (os *swiftObjectStore) Connect() error {
	return os.connect(&swift.Connection{
		UserName: os.conf[UsernameKey],
		ApiKey:   os.conf[PasswordKey],
		AuthUrl:  os.conf[AuthURLKey],
		Domain:   os.conf[DomainKey],
		Region:   os.conf[RegionKey],
		// TenantId: os.conf[ProjectKey],
	})
}

func (os *swiftObjectStore) connect(conn *swift.Connection) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if conn == nil {
		return errors.New("conn argument is nil")
	}

	// make deployment outside SL work for local env
	replaceSwiftObjectStoreURL(conn)

	// authenticate via retry logic
	var nerr error // non-recoverable error
	retry(10, 100*time.Millisecond, "connect", logr, func() error {
		err := conn.Authenticate()
		// only retry on timeout or bad request
		if err == swift.TimeoutError || err == swift.BadRequest {
			logr.Warnf("ObjectStore timeout/bad request error in connect(): %s. Retrying ...", err.Error())
			return err
		} else if err != nil {
			// non-retryable error
			logr.WithError(err).Errorf("ObjectStore connection error")
			nerr = err
		}
		if err == nil && conn.Authenticated() {
			os.conn = conn
		}
		return nil
	})
	return nerr
}

func (os *swiftObjectStore) UploadArchive(container string, object string, payload []byte) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.conn == nil {
		return ErrNotConnected
	}

	info, _, _ := os.conn.Container(container)

	if info.Name == "" { // container does not exist
		logr.Debugf("Creating storage container: %s\n", container)
		if err := os.conn.ContainerCreate(container, nil); err != nil {
			logr.WithError(err).Errorf("Storage container creation failed: %s", err.Error())
			return err
		}
	}
	return os.conn.ObjectPutBytes(container, object, payload, "application/zip")
}

func (os *swiftObjectStore) DownloadArchive(container string, object string) ([]byte, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))

	if os.conn == nil {
		return nil, ErrNotConnected
	}

	var payload []byte
	err := retry(10, 100*time.Millisecond, "ObjectGetBytes", logr, func() error {
		var err error
		payload, err = os.conn.ObjectGetBytes(container, object)
		return err
	})
	if err != nil {
		logr.WithError(err).Errorf("DownloadArchive failed %s, %s", container, object)
		return nil, err
	}
	return payload, nil
}


func (os *swiftObjectStore) DeleteArchive(container string, object string) error {
	if os.conn == nil {
		return ErrNotConnected
	}
	// delete object and then container (if last object)
	err := os.conn.ObjectDelete(container, object)
	if err != nil {
		log.WithError(err).Errorf("Deleting object %s in container %s failed", container, object)
		return err
	}

	// get container
	c, _, err := os.conn.Container(container)
	if err != nil {
		log.WithError(err).Errorf("Checking for container %s failed", container)
		return err
	}
	if c.Count == 0 {
		err = os.conn.ContainerDelete(container)
		if err != nil {
			log.WithError(err).Errorf("Deleting container %s failed", container)
		}
		return err
	}
	return nil
}

func (os *swiftObjectStore) GetTrainedModelSize(path string, numLearners int32) (int64, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.conn == nil {
		return 0,ErrNotConnected
	}

	// set default numLearners if not provided
	if numLearners <= 0 {
		numLearners = 1
	}

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
	trainedModelSize = 0

	//Iterate over all learner objects
	for iter := 0; iter < int(numLearners); iter++ {

		pathToLearner := objectPrefix + "/learner-" + strconv.Itoa(iter+1)
		objects, err := os.conn.Objects(container, &swift.ObjectsOpts{
			Path: pathToLearner,
		})
		logr.Debugf("objects: %s", objects)

		if err != nil {
			logr.WithError(err).Errorf("Checking object in container %s failed", container)
			return 0, err
		}

		for _, obj := range objects {
			logr.Debugf("trained model object: %s, %d", obj.Name, obj.Bytes)
			trainedModelSize += obj.Bytes
		}
	}
	return trainedModelSize, nil

}

func (os *swiftObjectStore) DownloadTrainedModelAsZipStream(path string, numLearners int32, writer io.Writer) error {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if os.conn == nil {
		return ErrNotConnected
	}

	// separate path in container and object

	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}
	if objectPrefix != "" {
		// Best effort to get the training ID
		logr = logr.WithField(logger.LogkeyTrainingID, objectPrefix)
	}
	logr.Debugf("Number of learners is: %d", numLearners)

	// Create a buffer to write our zip to
	w := zip.NewWriter(writer)

	var objects []swift.Object
	err := retry(10, 100*time.Millisecond, "Objects", logr, func() error {
		var err error
		var objects1 []swift.Object
		objects1, err = os.conn.Objects(container, &swift.ObjectsOpts{Path: objectPrefix})
		if err != nil {
			return err
		}
		// sometimes we don't get all the objects.  Call twice, and verify that the size is the same
		runtime.Gosched()
		objects, err = os.conn.Objects(container, &swift.ObjectsOpts{Path: objectPrefix})
		if err != nil {
			return err
		}
		if len(objects1) != len(objects) {
			err = errors.New("object count mismatch")
		}
		return err
	})
	if err != nil {
		logr.WithError(err).Errorf("Getting objects in container %s failed", container)
		return err
	}


	logr.Debugf("objects: %s", objects)
	if err != nil {
		logr.WithError(err).Errorf("Getting objects in container %s failed", container)
		return err
	}
	for _, obj := range objects {
		logr.Debugf("Downloading trained model object: %s, %d", obj.Name, obj.Bytes)
		f, err := w.Create(obj.Name)
		if err != nil {
			return err
		}

		err = retry(10, 100*time.Millisecond, "ObjectGet", logr, func() error {
			_, err := os.conn.ObjectGet(container, obj.Name, f, true, nil)
			return err
		})
		if err != nil {
			logr.WithError(err).Errorf("Getting object %s in container %s failed", obj.Name, container)
			return err
		}
	}


	if numLearners == 0 {
		numLearners = 1
		logr.Debugf("Forcing number learners to 1: %d", numLearners)
	}

	//Iterate over all learner objects
	for iter := 0; iter < int(numLearners); iter++ {

		pathToLearner := objectPrefix + "/learner-" + strconv.Itoa(iter+1)
		logr.Debugf("pathToLearner: %s", pathToLearner)

		err = retry(10, 100*time.Millisecond, "Objects", logr, func() error {
			var err error
			var objects1 []swift.Object
			objects1, err = os.conn.Objects(container, &swift.ObjectsOpts{Path: pathToLearner})
			if err != nil {
				return err
			}
			// sometimes we don't get all the objects.  Call twice, and verify that the size is the same
			runtime.Gosched()
			objects, err = os.conn.Objects(container, &swift.ObjectsOpts{Path: pathToLearner})
			if err != nil {
				return err
			}
			if len(objects1) != len(objects) {
				err = errors.New("object count mismatch")
			}
			return err
		})
		logr.Debugf("objects: %s", objects)

		if err != nil {
			logr.WithError(err).Errorf("Getting objects in container %s failed", container)
			return err
		}

		for _, obj := range objects {
			logr.Debugf("Downloading trained model object: %s, %d", obj.Name, obj.Bytes)

			f, err := w.Create(obj.Name)
			if err != nil {
				logr.WithError(err).Errorf("Creating ZIP file entry for object %s failed", obj.Name)
				return err
			}

			err = retry(10, 100*time.Millisecond, "ObjectGet", logr, func() error {
				_, err = os.conn.ObjectGet(container, obj.Name, f, true, nil)
				return err
			})
			if err != nil {
				logr.WithError(err).Errorf("Getting object %s in container %s failed", obj.Name, container)
				return err
			}
		}

	}
	if err := w.Close(); err != nil {
		logr.WithError(err).Errorf("Closing ZIP stream failed")
		return err
	}
	return nil
}

func (os *swiftObjectStore) DownloadTrainedModelLogFile(path string, numLearners int32, learnerIndex int32,
	objectPath string, outputWriter io.Writer) error {

	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))

	if os.conn == nil {
		return ErrNotConnected
	}

	// separate path in container and object
	var container, objectPrefix string
	index := strings.Index(path, "/")
	if index < 0 {
		container = path
	} else {
		container = path[0:index]
		objectPrefix = path[index+1:]
	}
	if objectPrefix != "" {
		// Best effort to get the training ID
		logr = logr.WithField(logger.LogkeyTrainingID, objectPrefix)
	}
	logr.Debugf("swiftObjectStore DownloadTrainedModelLogFile: entry: %s, %s", path, objectPath)
	logr.Debugf("Number of learners is: %d", numLearners)

	pathToLearner := objectPrefix + "/learner-" + strconv.Itoa(int(learnerIndex))
	objects, err := os.conn.Objects(container, &swift.ObjectsOpts{
		Path: pathToLearner,
	})
	logr.Debugf("objects: %s", objects)

	if err != nil {
		logr.WithError(err).Errorf("Getting objects in container %s failed", container)
		return err
	}

	fullObjectPath := pathToLearner + "/" + objectPath
	foundIt := false

	for _, obj := range objects {
		logr.Debugf("Found trained model object: %s, %d", obj.Name, obj.Bytes)
		logr.Debugf("Looking for: %s", fullObjectPath)

		if fullObjectPath == obj.Name {
			logr.Debugf("DownloadTrainedModelLogFile: Downloading trained model object: %s, %d", obj.Name, obj.Bytes)
			_, err = os.conn.ObjectGet(container, obj.Name, outputWriter, true, nil)
			if err != nil {
				logr.WithError(err).Errorf("Getting object %s in container %s failed", obj.Name, container)
				return err
			}
			foundIt = true
			break
		}
	}

	if !foundIt {
		return fmt.Errorf("DownloadTrainedModelLogFile: Could not find object: %s", fullObjectPath)
	}
	return nil
}

func (os *swiftObjectStore) ContainerExists(name string) (bool, error) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))

	if os.conn == nil {
		logr.Debugln("ContainerExists: object store not connected")
		return false, ErrNotConnected
	}

	// logr.Debugf("connection: %v", os.conn)

	var nerr error
	var exists bool
	// retry because of flakiness
	retry(10, 100*time.Millisecond, "ContainerExists", logr, func() error {
		_, _, err := os.conn.Container(name)
		if err == swift.TimeoutError || err == swift.BadRequest { // retryable error
			logr.Warnf("ObjectStore timeout/bad request error in ContainerExists: %s. Retrying ...", err.Error())
			return err
		} else if err == swift.ContainerNotFound {
			logr.Debugf("Container %s does not exist.", name)
		}
		nerr = err
		exists = err != swift.ContainerNotFound
		return nil
	})
	if nerr != nil {
		logr.Debugf("ContainerExists failed: %s", nerr.Error())
	}
	return exists, nil
}

func (os *swiftObjectStore) Disconnect() {
	if os.conn != nil {
		os.conn.UnAuthenticate()
	}
}

// replaceSwiftObjectStoreURL depending on the DLAAS_ENV variables we need to
// "rewrite" the object store URL to either use the internal SL proxy or the
// external URLs. This needs to work for both SL and Bluemix ObjectStore.
func replaceSwiftObjectStoreURL(conn *swift.Connection) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	env := viper.GetString(config.EnvKey)

	switch env {
	case config.DevelopmentEnv:
		replaceSwiftConnURL(conn, intDevBMObjectStoreURL)
	case config.StagingEnv:
		replaceSwiftConnURL(conn, intStagingBMObjectStoreURL)
	case config.ProductionEnv:
		replaceSwiftConnURL(conn, intProdBMObjectStoreURL)
	default: // any other env will use local settings (assuming outside SL)
		if strings.Contains(conn.AuthUrl, intSLObjectStoreURLFragment) {
			newURL := strings.Replace(conn.AuthUrl, intSLObjectStoreURLFragment, extSLObjectStoreURLFragment, 1)
			logr.Debugf("Replaced ObjectStore URL from %s to %s", conn.AuthUrl, newURL)
			conn.AuthUrl = newURL
		} else if strings.Contains(conn.AuthUrl, intDevBMObjectStoreURL) ||
			strings.Contains(conn.AuthUrl, intStagingBMObjectStoreURL) ||
			strings.Contains(conn.AuthUrl, intProdBMObjectStoreURL) {
			logr.Debugf("Replaced ObjectStore URL from %s to %s", conn.AuthUrl, extBMObjectStoreURL)
			conn.AuthUrl = extBMObjectStoreURL
		}
	}
}

func replaceSwiftConnURL(conn *swift.Connection, bluemixOSURL string) {
	logr := logger.LocLogger(log.StandardLogger().WithField("module", "storage"))
	if strings.Contains(conn.AuthUrl, extSLObjectStoreURLFragment) { // external SL URL part
		newURL := strings.Replace(conn.AuthUrl, extSLObjectStoreURLFragment, intSLObjectStoreURLFragment, 1)
		logr.Debugf("Replaced ObjectStore URL from %s to %s", conn.AuthUrl, newURL)
		conn.AuthUrl = newURL
	} else if strings.Contains(conn.AuthUrl, extBMObjectStoreURL) { // external BM
		logr.Debugf("Replaced ObjectStore URL from %s to %s", conn.AuthUrl, bluemixOSURL)
		conn.AuthUrl = bluemixOSURL
		conn.Internal = true // we need to use the internal IP that we get back from the auth service
	} else if strings.Contains(conn.AuthUrl, bluemixOSURL) {
		conn.Internal = true
	}
}
