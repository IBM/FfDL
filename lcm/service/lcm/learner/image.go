package learner

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	framework "github.com/IBM/FfDL/commons/framework"
)

//Image ...
type Image struct {
	Framework, Version, Tag, Registry, Namespace string
}

var learnerConfigPath = "/etc/learner-config-json/learner-config.json"

//GetLearnerImageForFramework returns the full route for the learner image
func GetLearnerImageForFramework(image Image) string {
	var learnerImage string
	if image.Registry != "" && image.Namespace != "" {
		learnerImage = fmt.Sprintf("%s/%s/%s:%s", image.Registry, image.Namespace, image.Framework, image.Version)
	} else {
		learnerTag := getLearnerTag(image.Framework, image.Version, image.Tag)
		dockerRegistry := viper.GetString(config.LearnerRegistryKey)
		learnerImage = fmt.Sprintf("%s/%s_gpu_%s:%s", dockerRegistry, image.Framework, image.Version, learnerTag)
	}
	return learnerImage
}

func getLearnerTag(frameworkVersion, version, learnerTagFromRequest string) string {

	learnerTag := viper.GetString(config.LearnerTagKey)
	// Use any tag in the request (ie, specified in the manifest)
	learnerImageTagInManifest := learnerTagFromRequest
	if "" == learnerImageTagInManifest {
		// not in request; try looking up from configmap/learner-config
		imageBuildTag := framework.GetImageBuildTagForFramework(frameworkVersion, version, learnerConfigPath)
		if imageBuildTag != "" {
			return imageBuildTag
		}
	} else {
		learnerTag = learnerImageTagInManifest
	}
	return learnerTag
}
