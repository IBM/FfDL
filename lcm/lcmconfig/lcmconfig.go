package lcmconfig

import (
	"github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
	k8srest "k8s.io/client-go/rest"
	"github.com/IBM/FfDL/commons/logger"
)

// GetKubernetesConfig returns the configuration to connect to a Kubernetes cluster.
// If the URL is empty, then use the InClusterConfig.
// Otherwise, get the CA cert
func GetKubernetesConfig() *k8srest.Config {
	host := config.GetLearnerKubeURL()
	var c *k8srest.Config
	if host == "" {
		c, _ = k8srest.InClusterConfig()
	} else {
		c = &k8srest.Config{
			Host: host,
			TLSClientConfig: k8srest.TLSClientConfig{
				CAFile: config.GetLearnerKubeCAFile(),
			},
		}
		token := config.GetLearnerKubeToken()
		if token == "" {
			tokenFileContents := config.GetFileContents(config.GetLearnerKubeTokenFile())
			if tokenFileContents != "" {
				token = tokenFileContents
			}
		}
		if token == "" {
			c.TLSClientConfig.KeyFile = config.GetLearnerKubeKeyFile()
			c.TLSClientConfig.CertFile = config.GetLearnerKubeCertFile()
		} else {
			c.BearerToken = token
		}
	}
	return c
}

//GetImagePullPolicy image pull policy if set else v1core.PullAlways
func GetImagePullPolicy() v1core.PullPolicy {

	policy := v1core.PullPolicy(config.GetString( config.ImagePullPolicy))
	logr := logger.LocLogger(logrus.StandardLogger().WithField("module", "lcm"))

	logr.Debugf("pull policy from logrus is: %s", policy)

	switch policy {
	case v1core.PullAlways, v1core.PullIfNotPresent, v1core.PullNever:
		logr.Infof("policy specified for pulling images %s", policy)
		return policy
	}
	return v1core.PullIfNotPresent
}
