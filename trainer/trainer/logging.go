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

package trainer

import (
	log "github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/logger"
)

func logEntry() *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyTrainerService)
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWithTraining(trainingID string) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyTrainerService)
	data[logger.LogkeyTrainingID] = trainingID
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}

func logWith(trainingID string, userID string) *log.Entry {
	data := logger.NewDlaaSLogData(logger.LogkeyTrainerService)
	data[logger.LogkeyTrainingID] = trainingID
	data[logger.LogkeyUserID] = userID
	return &log.Entry{Logger: log.StandardLogger(), Data: data}
}
