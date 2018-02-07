package lcmconfig

import (
	"github.ibm.com/ffdl/ffdl-core/commons/config"
	k8srest "k8s.io/client-go/rest"
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
