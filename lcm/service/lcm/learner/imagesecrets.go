package learner

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/commons/service"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type dockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// GenerateImagePullSecret ... creates secret only for custom images; otherwise returns default secret name
func GenerateImagePullSecret(k8sClient kubernetes.Interface, req *service.JobDeploymentRequest) (string, error) {

	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)

	// if no custom image, then use our default pull secret
	if req.ImageLocation == nil {
		return imagePullSecret, nil
	}

	// if no token specified, then use ours
	if req.ImageLocation.AccessToken == "" {
		return "", errors.New("Custom image access token is missing")
	}

	// build a custom secret
	imagePullSecret = "customimage-" + req.Name
	trainingID := req.TrainingId
	server := req.ImageLocation.Registry
	token := req.ImageLocation.AccessToken
	email := req.ImageLocation.Email
	// format of the .dockercfg entry
	entry := make(map[string]dockerConfigEntry)
	entry[server] = dockerConfigEntry{
		Username: "token",
		Password: token,
		Email:    email,
		Auth:     base64.StdEncoding.EncodeToString([]byte("token:" + token)),
	}
	dockerCfgContent, _ := json.Marshal(entry)
	// create Secret object
	secret := v1core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePullSecret,
			Namespace: config.GetLearnerNamespace(),
			Labels:    map[string]string{"training_id": trainingID}, // this makes sure the secret is deleted with the other learner components
		},
		Type: v1core.SecretTypeDockercfg, // kubernetes.io/dockercfg
		Data: map[string][]byte{},
	}
	// add the dockercfg content (as binary)
	secret.Data[v1core.DockerConfigKey] = dockerCfgContent
	// create the secret
	if _, err := k8sClient.CoreV1().Secrets(secret.Namespace).Create(&secret); err != nil {
		return imagePullSecret, err
	}

	// return its name
	return imagePullSecret, nil
}
