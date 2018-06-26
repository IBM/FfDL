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

package jobmonitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
)

func init() {
	config.InitViper()
}

func TestTransitions(t *testing.T) {

	jm := &JobMonitor{
		TrainingID:  "unit-test-trainingId",
		UserID:      "unit-test-userId",
		NumLearners: 1,
		JobName:     "unit-test-jobName",
		trMap:       initTransitionMap(),
	}

	assert.EqualValues(t, true, jm.isTransitionAllowed("PENDING", "DOWNLOADING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "PROCESSING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "STORING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "STORING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "COMPLETED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("DOWNLOADING", "HALTED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("PROCESSING", "PROCESSING"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "FAILED"))
	assert.EqualValues(t, true, jm.isTransitionAllowed("STORING", "HALTED"))

	assert.EqualValues(t, false, jm.isTransitionAllowed("STORING", "DOWNLOADING"))
	assert.EqualValues(t, false, jm.isTransitionAllowed("COMPLETED", "PROCESSING"))
	assert.EqualValues(t, false, jm.isTransitionAllowed("FAILED", "COMPLETED"))

}
