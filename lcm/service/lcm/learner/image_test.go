package learner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {

	learnerConfigPath = "../../../testdata/learner-config.json" //Uses configmap in testdata directory

}

func TestGetImageNameWithCustomRegistry(t *testing.T) {
	image := Image{
		Framework: "tensorflow",
		Version:   "1.5",
		Tag:       "latest",
		Registry:  "registry.ng.bluemix.net",
		Namespace: "custom_reg",
	}

	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/custom_reg/tensorflow:1.5", learnerImage)
}

func TestGetValidImageName(t *testing.T) {
	image := Image{
		Framework: "tensorflow",
		Version:   "1.5",
		Tag:       "latest",
	}
	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/dlaas_dev/tensorflow_gpu_1.5:latest", learnerImage)

}

func TestGetInvalidImageName(t *testing.T) {
	image := Image{
		Framework: "dlaas_config_test",
		Version:   "2.2",
		Tag:       "",
	}
	learnerImage := GetLearnerImageForFramework(image)
	assert.Equal(t, "registry.ng.bluemix.net/dlaas_dev/dlaas_config_test_gpu_2.2:master-2", learnerImage)

}
