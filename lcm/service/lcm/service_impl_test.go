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

package lcm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"

	"github.com/IBM/FfDL/commons/service"
	// "github.com/coreos/etcd/clientv3"
	// "github.com/IBM/FfDL/lcm/coord"
)

func init() {
	config.InitViper()
}

func TestCalcMemory(t *testing.T) {
	r := &service.ResourceRequirements{
		Memory:     512,
		MemoryUnit: service.ResourceRequirements_MB,
	}
	assert.EqualValues(t, 512, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     8.5,
		MemoryUnit: service.ResourceRequirements_GB,
	}
	assert.EqualValues(t, 8500, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     128,
		MemoryUnit: service.ResourceRequirements_MiB,
	}
	assert.EqualValues(t, 134.22, calcMemory(r))

	r = &service.ResourceRequirements{
		Memory:     8.2,
		MemoryUnit: service.ResourceRequirements_GiB,
	}
	assert.EqualValues(t, 8804.68, calcMemory(r))

}
