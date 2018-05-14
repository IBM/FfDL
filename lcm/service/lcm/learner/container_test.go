package learner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1core "k8s.io/api/core/v1"
	v1resource "k8s.io/apimachinery/pkg/api/resource"
)

func TestContainerWithMountedCOS(t *testing.T) {

	var envars []v1core.EnvVar
	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)

	container := Container{
		Image: Image{Framework: "tensorflow", Version: "1.5", Tag: "latest"},
		Resources: Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: []v1core.VolumeMount{v1core.VolumeMount{MountPath: "/nfs"},
			v1core.VolumeMount{MountPath: "/nfs"}, v1core.VolumeMount{MountPath: "/cos/data"}, v1core.VolumeMount{MountPath: "/cos/results"}},
		Name:    "test-learner-container",
		EnvVars: envars,
		Command: "echo hello",
	}

	containerCreated := CreateContainerSpec(container)
	assert.Equal(t, len(containerCreated.VolumeMounts), 4)

}

func TestContainerWithNoMountedCOS(t *testing.T) {

	var envars []v1core.EnvVar
	cpuCount := v1resource.NewMilliQuantity(int64(float64(1)*1000.0), v1resource.DecimalSI)
	gpuCount := v1resource.NewQuantity(int64(1), v1resource.DecimalSI)
	memCount := v1resource.NewQuantity(1024, v1resource.DecimalSI)

	container := Container{
		Image: Image{Framework: "tensorflow", Version: "1.5", Tag: "latest"},
		Resources: Resources{
			CPUs: *cpuCount, Memory: *memCount, GPUs: *gpuCount,
		},
		VolumeMounts: []v1core.VolumeMount{v1core.VolumeMount{MountPath: "/nfs"},
			v1core.VolumeMount{MountPath: "/nfs"}},
		Name:    "test-learner-container",
		EnvVars: envars,
		Command: "echo hello",
	}

	containerCreated := CreateContainerSpec(container)
	assert.Equal(t, len(containerCreated.VolumeMounts), 2)

}
