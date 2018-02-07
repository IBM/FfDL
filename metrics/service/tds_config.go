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

package service

import (
	"sync"
	"github.com/spf13/viper"
)

const (
	// envPrefix is the DLaaS prefix that viper uses for prefixing env variables (it is used upper case).
	envPrefix = "tds"

	// TdsDebug is the viper key to enable extended debug-session style logging in the TDS.
	TdsDebug = "tds_debug"
)

var viperInitOnce sync.Once

func tdsInit() {
	InitViper()
}

// InitViper is initializing the configuration system
func InitViper() {
	viperInitOnce.Do(func() {
		viper.SetDefault(TdsDebug, false)
	})
}
