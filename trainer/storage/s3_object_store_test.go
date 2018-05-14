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
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/IBM/FfDL/commons/config"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	testingBucketKey     = "objectstore.bucket.testing"
	defaultTestingBucket = "dlaas-testing"
)

func TestS3Connect(t *testing.T) {
	os, err := NewS3ObjectStore(config.GetDataStoreConfig())
	assert.NoError(t, err)
	assert.NotNil(t, os)
	assert.NoError(t, os.Connect())

	// TODO figure out how to test a func that fatals
	// _, err = NewS3ObjectStore(nil)
	// assert.Error(t, err)
	//
	// _, err = NewS3ObjectStore(make(map[string]string))
	// assert.Error(t, err)
}

func TestS3GetRegion(t *testing.T) {
	os := &s3ObjectStore{
		conf: config.GetDataStoreConfig(),
	}
	// default value
	assert.Equal(t, "us-standard", os.getRegion())

	m := make(map[string]string, 1)
	m[RegionKey] = "thailand"
	os = &s3ObjectStore{
		conf: m,
	}
	assert.Equal(t, "thailand", os.getRegion())
}

func TestS3ContainerExists(t *testing.T) {
	os, _ := NewS3ObjectStore(config.GetDataStoreConfig())
	os.Connect()
	defer os.Disconnect()

	// test non-existing bucket
	exists, err := os.ContainerExists(fmt.Sprintf("test-%d", time.Now().Unix()))
	assert.NoError(t, err)
	assert.False(t, exists)

	// test existing bucket
	exists, err = os.ContainerExists(getTestingBucket())
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestS3Archive(t *testing.T) {
	os, _ := NewS3ObjectStore(config.GetDataStoreConfig())
	os.Connect()
	defer os.Disconnect()

	// upload archive
	container := fmt.Sprintf("test-bucket-%d", time.Now().UnixNano())
	objectID := "sample_nlc_model.zip"

	zipBytes := getZipSample(t)
	err := os.UploadArchive(container, objectID, zipBytes)
	assert.NoError(t, err)

	payload, err := os.DownloadArchive(container, objectID)
	assert.NoError(t, err)
	// log.Debugf("zip sample size: ", len(zipBytes))
	// log.Debugf("payload size: ", len(payload))
	assert.EqualValues(t, zipBytes, payload)

	err = os.DeleteArchive(container, objectID)
	assert.NoError(t, err)
}

func TestS3DownloadTrainedModelAsZipStream(t *testing.T) {
	os, _ := NewS3ObjectStore(config.GetDataStoreConfig())
	os.Connect()
	defer os.Disconnect()

	// TODO upload sample - currently we use one present on object store

	buff := bytes.NewBuffer(nil)
	path := fmt.Sprintf("%s/training-unittest", getTestingBucket())
	err := os.DownloadTrainedModelAsZipStream(path, 2, buff)
	assert.NoError(t, err)
	assert.NotZero(t, buff.Bytes())
	assertInZip(t, buff.Bytes(), "training-unittest/learner-1/model.tf")
	assertInZip(t, buff.Bytes(), "training-unittest/learner-1/training-logs.txt")
	assertInZip(t, buff.Bytes(), "training-unittest/learner-2/model.tf")
	assertInZip(t, buff.Bytes(), "training-unittest/learner-2/training-log.txt")
}

func TestS3DownloadTrainedModelLogFile(t *testing.T) {
	os, _ := NewS3ObjectStore(config.GetDataStoreConfig())
	os.Connect()
	defer os.Disconnect()

	var b []byte
	buff := bytes.NewBuffer(b)
	bucket := getTestingBucket()
	path := fmt.Sprintf("%s/training-unittest", bucket)
	err := os.DownloadTrainedModelLogFile(path, 0, 1, "training-logs.txt", buff)
	assert.NoError(t, err)
	assert.NotZero(t, buff.Bytes())

	err = os.DownloadTrainedModelLogFile(bucket, 0, 1, "training-unittest/doesnotexist/training-logs.txt", buff)
	assert.Error(t, err)
}
func getTestingBucket() string {
	if viper.IsSet(testingBucketKey) {
		return viper.GetString(testingBucketKey)
	}
	return defaultTestingBucket
}
