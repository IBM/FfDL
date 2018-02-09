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

package util

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/IBM/FfDL/commons/logger"
)

func init() {
	logger.Config()
}

func TestZipUnzipRountrip(t *testing.T) {
	filename, _ := filepath.Abs("./testdata")
	zipFilename := path.Join(os.TempDir(), "zipper-test.zip")
	defer os.Remove(zipFilename)

	// create the zip
	err := Zip(filename, zipFilename)

	assert.NoError(t, err)
	fi, err2 := os.Stat(zipFilename)
	assert.NoError(t, err2)
	assert.Contains(t, zipFilename, fi.Name())

	// unzip
	targetDir := path.Join(os.TempDir(), "output-zip-dir")
	defer os.RemoveAll(targetDir)

	err = Unzip(zipFilename, targetDir)
	assert.NoError(t, err)

	fiOrig, _ := os.Stat(filename)
	fi, err2 = os.Stat(path.Join(os.TempDir(), "output-zip-dir", path.Base(filename)))
	assert.NoError(t, err2)

	assert.Contains(t, filename, fi.Name())
	assert.Equal(t, fi.Size(), fiOrig.Size())
}

func TestZipUnzipByteArray(t *testing.T) {
	filename, _ := filepath.Abs("./testdata")

	// zip
	archive, errZip := ZipToBytes(filename)
	assert.NoError(t, errZip)
	assert.NotZero(t, len(archive))

	// unzip from bytes
	targetDir := path.Join(os.TempDir(), "output-zip-dir2")
	defer os.RemoveAll(targetDir)

	errUnzip := UnzipFromBytes(archive, targetDir)
	assert.NoError(t, errUnzip)

	fiOrig, _ := os.Stat(filename)
	fi, err2 := os.Stat(path.Join(targetDir, path.Base(filename)))
	assert.NoError(t, err2)

	assert.Contains(t, filename, fi.Name())
	assert.Equal(t, fi.Size(), fiOrig.Size())

}
