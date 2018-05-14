package helper

import (
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	v1beta1 "k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreatePodSpec ...
func CreatePodSpec(containers []v1core.Container, volumes []v1core.Volume, labels map[string]string) v1core.PodTemplateSpec {
	labels["service"] = "dlaas-lhelper" //controls ingress/egress
	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)
	automountSeviceToken := false
	return v1core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: v1core.PodSpec{
			Containers: containers,
			Volumes:    volumes,
			ImagePullSecrets: []v1core.LocalObjectReference{
				v1core.LocalObjectReference{
					Name: imagePullSecret,
				},
			},
			AutomountServiceAccountToken: &automountSeviceToken,
		},
	}
}

//CreateDeploymentForHelper ...
func CreateDeploymentForHelper(name string, podTemplateSpec v1core.PodTemplateSpec) *v1beta1.Deployment {

	revisionHistoryLimit := int32(0) //https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#clean-up-policy

	//TODO consider this as well https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#progress-deadline-seconds
	//but not sure if we can nicely revert back
	return &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.DeploymentSpec{
			Template:             podTemplateSpec,
			RevisionHistoryLimit: &revisionHistoryLimit, //we never rollback these
		},
	}
}
