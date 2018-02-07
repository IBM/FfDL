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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemObjectStore(t *testing.T) {
	os, _:= NewInMemObjectStore(nil)

	payload := []byte{0x50, 0x31, 0x13}
	container := "foo"
	object := "obj1"

	os.UploadArchive(container, object, payload)
	download, _ := os.DownloadArchive(container, object)
	assert.EqualValues(t, download, payload)

	os.DeleteArchive(container, object)

	download, err := os.DownloadArchive(container, object)
	assert.Error(t, err)
	assert.Nil(t, download)
}

func TestDownloadTrainedModelAsZipFake(t *testing.T) {

	ostore, _ := NewInMemObjectStore(nil)
	ostore.Connect()
	defer ostore.Disconnect()

	ostore.UploadArchive("foobar", "training-foo/pseudo.model", []byte("my super model"))
	ostore.UploadArchive("foobar", "training-foo/training-log.txt", []byte("all great"))
	defer ostore.DeleteArchive("foobar", "training-foo/pseudo.model")
	defer ostore.DeleteArchive("foobar", "training-foo/training-log.txt")

	buf := new(bytes.Buffer)
	err := ostore.DownloadTrainedModelAsZipStream("foobar/training-foo", 1, buf)
	assert.NoError(t, err)
	assert.NotEmpty(t, buf.Bytes())

	// TODO should unzip and check
}
