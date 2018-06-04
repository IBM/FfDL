package lcm

import (
	"github.com/IBM/FfDL/commons/service"
	"github.com/spf13/viper"

	v1core "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"os"
	"fmt"
)

const learnerEntrypointFilesVolume = "learner-entrypoint-files"
const learnerEntrypointFilesPath = "/entrypoint-files"
const hostDataMountVolume = "mounted-host-data"
const hostDataMountPath = "/host-data"
const imagePrefixKey = "image.prefix"
const mountHostDataKey = "mount.host.data"


func extendLearnerContainer(learner *v1core.Container, req *service.JobDeploymentRequest) {

	learnerImage := "__unknown__"

	switch req.Framework {
	case caffeFrameworkName:
		learnerImage = "bvlc/caffe:" + req.Version
	case tfFrameworkName:
		learnerImage = "tensorflow/tensorflow:" + req.Version
	case caffe2FrameworkName:
		learnerImage = "caffe2ai/caffe2:" + req.Version
	case pytorchFrameworkName:
		learnerImage = "pytorch/pytorch:" + req.Version
	case h2o3FrameworkName:
		learnerImage = "opsh2oai/h2o3-ffdl:" + req.Version
	case customFrameworkName:
		learnerImage = req.Version
	default:
		// TODO!
	}

	extCmd := "export PATH=/usr/local/bin/:$PATH; cp " + learnerEntrypointFilesPath + "/*.sh /usr/local/bin/; chmod +x /usr/local/bin/*.sh;"
	extMount := v1core.VolumeMount{
		Name:      learnerEntrypointFilesVolume,
		MountPath: learnerEntrypointFilesPath,
	}
	learner.VolumeMounts = append(learner.VolumeMounts, extMount)

	if doMountHostData() {
		hostMount := v1core.VolumeMount{
			Name:      hostDataMountVolume,
			MountPath: hostDataMountPath,
		}
		learner.VolumeMounts = append(learner.VolumeMounts, hostMount)
	}

	learner.Image = learnerImage
	learner.Command[2] = extCmd + learner.Command[2]
}

func extendLearnerDeployment(deployment *v1beta1.Deployment) {

	// learner entrypoint files volume
	learnerEntrypointFilesVolume := v1core.Volume{
		Name: learnerEntrypointFilesVolume,
		VolumeSource: v1core.VolumeSource{
			ConfigMap: &v1core.ConfigMapVolumeSource{
				LocalObjectReference: v1core.LocalObjectReference{
					Name: learnerEntrypointFilesVolume,
				},
			},
		},
	}
	deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, learnerEntrypointFilesVolume)

	if doMountHostData() {
		hostDataVolume := v1core.Volume{
			Name: hostDataMountVolume,
			VolumeSource: v1core.VolumeSource{
				HostPath: &v1core.HostPathVolumeSource{
					Path: hostDataMountPath,
				},
			},
		}
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, hostDataVolume)
	}
}

func dataBrokerImageNameExtended(dockerRegistry string, dataBrokerType string, dataBrokerTag string) string {
	imagePrefix := getImagePrefix()
	return fmt.Sprintf("%s/%sdatabroker_%s:%s", dockerRegistry, imagePrefix, dataBrokerType, dataBrokerTag)
}

func controllerImageNameExtended(dockerRegistry string, servicesTag string) string {
	imagePrefix := getImagePrefix()
	return fmt.Sprintf("%s/%scontroller:%s", dockerRegistry, imagePrefix, servicesTag)
}

func jobmonitorImageNameExtended(dockerRegistry string, jmTag string) string {
	imagePrefix := getImagePrefix()
	return fmt.Sprintf("%s/%sjobmonitor:%s", dockerRegistry, imagePrefix, jmTag)
}

func getImagePrefix() string {
	return viper.GetString(imagePrefixKey)
}

func doMountHostData() bool {
	mountPath := viper.GetString(mountHostDataKey)
	if mountPath == "1" || mountPath == "true" {
		return true
	}
	// return exists(hostDataMountPath) || true
	return false
}

func exists(path string) bool {
	_, err := os.Stat(hostDataMountPath)
	return err == nil
}
