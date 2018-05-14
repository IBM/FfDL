package lcmconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
	v1core "k8s.io/api/core/v1"
)

func TestGetImagePullPolicy(t *testing.T) {
	config.SetDefault("IMAGE_PULL_POLICY", "Always")
	assert.Equal(t, v1core.PullAlways, GetImagePullPolicy())

	config.SetDefault("IMAGE_PULL_POLICY", "UselessValue")
	assert.Equal(t, v1core.PullAlways, GetImagePullPolicy())

	config.SetDefault("IMAGE_PULL_POLICY", "IfNotPresent")
	assert.Equal(t, v1core.PullIfNotPresent, GetImagePullPolicy())

}
