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
	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//COSVolumeSecret ...
type COSVolumeSecret struct {
	ID, TrainingID, Username, APIKey string
}

//SSHVolumeSecret ...
type SSHVolumeSecret struct {
	ID, TrainingID, Framework, Version string
}

//Secrets ...
type Secrets struct {
	TrainingDataSecret *COSVolumeSecret
	ResultsDirSecret   *COSVolumeSecret
}

//CreateVolumeSecretsSpec ...
func CreateVolumeSecretsSpec(secrets Secrets) []*v1core.Secret {

	var secretSpecs []*v1core.Secret
	if secrets.TrainingDataSecret != nil {
		cosTrainingDataVolumeSecretParams := secrets.TrainingDataSecret
		secretSpecs = append(secretSpecs, generateCOSVolumeSecret(cosTrainingDataVolumeSecretParams.ID, cosTrainingDataVolumeSecretParams.TrainingID, cosTrainingDataVolumeSecretParams.Username, cosTrainingDataVolumeSecretParams.APIKey))
	}

	if secrets.ResultsDirSecret != nil {
		cosResultDirVolumeSecretParams := secrets.ResultsDirSecret
		secretSpecs = append(secretSpecs, generateCOSVolumeSecret(cosResultDirVolumeSecretParams.ID, cosResultDirVolumeSecretParams.TrainingID, cosResultDirVolumeSecretParams.Username, cosResultDirVolumeSecretParams.APIKey))
	}

	return secretSpecs
}

func generateCOSVolumeSecret(id, trainingID, username, apikey string) *v1core.Secret {
	// create secret
	spec := v1core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: config.GetLearnerNamespace(),
			Labels:    map[string]string{"training_id": trainingID},
		},
		Type: cosMountDriverName,
		StringData: map[string]string{
			"access-key": username,
			"secret-key": apikey,
		},
	}

	return &spec
}
