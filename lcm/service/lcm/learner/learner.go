package learner

import (
	"github.com/spf13/viper"
	"github.com/IBM/FfDL/commons/config"
	v1beta1 "k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreatePodSpec ...
func CreatePodSpec(containers []v1core.Container, volumes []v1core.Volume, labels map[string]string, nodeSelector map[string]string) v1core.PodTemplateSpec {
	labels["service"] = "dlaas-learner" //label that denies ingress/egress
	imagePullSecret := viper.GetString(config.LearnerImagePullSecretKey)
	automountSeviceToken := false
	return v1core.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Annotations: map[string]string{
				"scheduler.alpha.kubernetes.io/tolerations": `[ { "key": "dedicated", "operator": "Equal", "value": "gpu-task" } ]`,
				"scheduler.alpha.kubernetes.io/nvidiaGPU":   `{ "AllocationPriority": "Dense" }`,
			},
		},
		Spec: v1core.PodSpec{
			Containers: containers,
			Volumes:    volumes,
			ImagePullSecrets: []v1core.LocalObjectReference{
				v1core.LocalObjectReference{
					Name: imagePullSecret,
				},
			},
			Tolerations: []v1core.Toleration{
				v1core.Toleration{
					Key:      "dedicated",
					Operator: v1core.TolerationOpEqual,
					Value:    "gpu-task",
					Effect:   v1core.TaintEffectNoSchedule,
				},
			},
			NodeSelector:                 nodeSelector,
			AutomountServiceAccountToken: &automountSeviceToken,
		},
	}
}

//CreateStatefulSetSpecForLearner ...
func CreateStatefulSetSpecForLearner(name, servicename string, replicas int, podTemplateSpec v1core.PodTemplateSpec) *v1beta1.StatefulSet {
	var replicaCount = int32(replicas)
	revisionHistoryLimit := int32(0) //https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#clean-up-policy

	return &v1beta1.StatefulSet{

		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: podTemplateSpec.Labels,
		},
		Spec: v1beta1.StatefulSetSpec{
			ServiceName:          servicename,
			Replicas:             &replicaCount,
			Template:             podTemplateSpec,
			RevisionHistoryLimit: &revisionHistoryLimit, //we never rollback these
			//PodManagementPolicy: v1beta1.ParallelPodManagement, //using parallel pod management in stateful sets to ignore the order. not sure if this will affect the helper pod since any pod in learner can come up now
		},
	}
}

//CreateServiceSpec ... this service will govern the statefulset
func CreateServiceSpec(name string, trainingID string) *v1core.Service {

	return &v1core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"training_id": trainingID,
			},
		},
		Spec: v1core.ServiceSpec{
			Selector: map[string]string{"training_id": trainingID},
			Ports: []v1core.ServicePort{
				v1core.ServicePort{
					Name:     "ssh",
					Protocol: v1core.ProtocolTCP,
					Port:     22,
				},
				v1core.ServicePort{
					Name:     "tf-distributed",
					Protocol: v1core.ProtocolTCP,
					Port:     2222,
				},
			},
			ClusterIP: "None",
		},
	}
}
