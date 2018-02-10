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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/util"
	"archive/zip"
	"bytes"
	"regexp"
	"math/rand"
)

func getZipSample(t *testing.T) []byte {
	zipBytes, err := util.ZipToBytes("testdata/nlc-model-demo")
	assert.NoError(t, err)
	assert.NotEmpty(t, zipBytes)
	return zipBytes
}

func assertInZip(t testing.TB, zippedBytes []byte, expectedFile string) {

	r, err := zip.NewReader(bytes.NewReader(zippedBytes), int64(len(zippedBytes)))
	assert.NoError(t, err)

	found := false
	fileSize := uint32(0)
	for _, f := range r.File {
		matched, err := regexp.MatchString(expectedFile, f.Name)
		if err != nil {
			assert.Fail(t, "expectedFile is not a valid regex")
			break
		}
		if matched {
			found = true
			fileSize = f.CompressedSize
			break
		}
	}
	assert.True(t, found, expectedFile+" not found in zip")
	assert.NotZero(t, fileSize, expectedFile+" has size 0")
}

func generateByteArr(len int) []byte {
	arr := make([]byte, len)
	rand.Read(arr)
	return arr
}
