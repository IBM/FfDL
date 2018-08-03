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

package learner

import (
	"path"
	"path/filepath"
	"strconv"
	"strings"

	v1core "k8s.io/api/core/v1"
)

//PopulateLearnerEnvVariablesAndLabels ... create envvars for learner from shared env vars. add learner specific envs vars + filter out what is not required
func PopulateLearnerEnvVariablesAndLabels(existingEnvVars []v1core.EnvVar, trainingID string, numLearners int, statefulsetName string, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []v1core.EnvVar {

	var envVars []v1core.EnvVar
	envVars = append(envVars, existingEnvVars...)

	envVars = append(envVars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAME", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	envVars = append(envVars, v1core.EnvVar{Name: "DOWNWARD_API_POD_NAMESPACE", ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.namespace"}}})

	//For now assuming sateful set name is same as service name
	envVars = append(envVars, v1core.EnvVar{Name: "LEARNER_NAME_PREFIX", Value: statefulsetName})

	envVars = append(envVars, v1core.EnvVar{Name: "TRAINING_ID", Value: trainingID})
	envVars = append(envVars, v1core.EnvVar{Name: "DLAAS_JOB_ID", Value: trainingID})
	envVars = append(envVars, v1core.EnvVar{Name: "NUM_LEARNERS", Value: strconv.Itoa(numLearners)})

	/*
	   learner ID is being set as a part of the command
	   	envVars = append(envVars, v1core.EnvVar{
	   		Name:      "LEARNER_ID",
	   		ValueFrom: &v1core.EnvVarSource{FieldRef: &v1core.ObjectFieldSelector{FieldPath: "metadata.name"}},
	   	})
	*/

	vars := generateLearnerContainerEnvVars(envVars, trainingID, mountTrainingDataStoreInLearner, mountResultsStoreInLearner)
	return vars

}

//FIXME for now not changing this much and just whitelisting rather than makign the list explicit
//need to make this function more testable
func generateLearnerContainerEnvVars(envVars []v1core.EnvVar, trainingID string, mountTrainingDataStoreInLearner, mountResultsStoreInLearner bool) []v1core.EnvVar {

	var whitelisted = map[string]struct{}{
		"MODEL_DIR":                  {},
		"DATA_DIR":                   {},
		"RESULT_DIR":                 {},
		"RESULT_BUCKET_DIR":          {},
		"LOG_DIR":                    {},
		"CHECKPOINT_DIR":             {},
		"JOB_STATE_DIR":              {},
		"TRAINING_JOB":               {},
		"TRAINING_COMMAND":           {},
		"TRAINING_ID":                {},
		"LEARNER_ID":                 {},
		"GPU_COUNT":                  {},
		"NUM_LEARNERS":               {},
		"LEARNER_NAME_PREFIX":        {},
		"DOWNWARD_API_POD_NAME":      {},
		"DOWNWARD_API_POD_NAMESPACE": {},
	}

	// Given a set of environment variables, return the subset that should appear in the learner container.
	getLearnerContainerEnvVars := func(allVars []v1core.EnvVar) []v1core.EnvVar {
		vars := make([]v1core.EnvVar, 0, 0)
		for _, ev := range allVars {
			if _, exists := whitelisted[ev.Name]; exists {
				vars = append(vars, ev)
			} else {
				// don't include this var.
			}
		}
		return vars
	}

	filteredVars := getLearnerContainerEnvVars(envVars)

	//argh!! this code was already there
	vars := make([]v1core.EnvVar, 0, len(filteredVars))
	var checkpointDir string
	var resultBucketDir string
	for _, ev := range filteredVars {
		if strings.HasSuffix(ev.Name, "_DIR") {
			var dir string
			if ev.Name == "DATA_DIR" && mountTrainingDataStoreInLearner {
				dir = filepath.Join("/mnt/data", ev.Value)
			} else if ev.Name == "RESULT_DIR" && mountResultsStoreInLearner {
				resultBucketDir = filepath.Join("/mnt/results", ev.Value)
				checkpointDir = filepath.Join(resultBucketDir, "_wml_checkpoints")
				dir = filepath.Join(resultBucketDir, trainingID)
			} else {
				dir = path.Join("/job", ev.Value) //FIXME stupid hack to add /job in front of all paths
			}
			vars = append(vars, v1core.EnvVar{Name: ev.Name, Value: dir})
		} else {
			vars = append(vars, ev)
		}
	}

	vars = append(vars,
		v1core.EnvVar{Name: "JOB_STATE_DIR", Value: "/job"},
		v1core.EnvVar{Name: "CHECKPOINT_DIR", Value: checkpointDir},
		v1core.EnvVar{Name: "RESULT_BUCKET_DIR", Value: resultBucketDir})
	return vars
}
