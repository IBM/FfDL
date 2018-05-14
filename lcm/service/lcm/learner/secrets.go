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
