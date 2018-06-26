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

package lcmconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
)

func TestGetImagePullPolicy(t *testing.T) {
	config.SetDefault("IMAGE_PULL_POLICY", "Always")
	assert.Equal(t, v1core.PullAlways, GetImagePullPolicy())

	config.SetDefault("IMAGE_PULL_POLICY", "UselessValue")
	assert.Equal(t, v1core.PullAlways, GetImagePullPolicy())

	config.SetDefault("IMAGE_PULL_POLICY", "IfNotPresent")
	assert.Equal(t, v1core.PullIfNotPresent, GetImagePullPolicy())

}
