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
	"fmt"
	"time"

	"github.com/IBM/FfDL/commons/logger"
)

//
// Helper retry function
//
func Retry(attempts int, interval time.Duration, description string, logr *logger.LocLoggingEntry, callback func() error) (err error) {
	for i := 0; ; i++ {
		err = callback()
		if err == nil {
			return nil
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(interval)
		logr.Warnf("Retrying function %s due to error %s", description, err)
	}
	return fmt.Errorf("function %s after %d attempts, last error: %s", description, attempts, err)
}
