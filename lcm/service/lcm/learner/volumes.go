package learner

import (
	//	"strconv"
	//	"github.com/IBM/FfDL/commons/logger"
	v1core "k8s.io/api/core/v1"
	log "github.com/sirupsen/logrus"
	"github.com/IBM/FfDL/commons/logger"
	yaml "gopkg.in/yaml.v2"
)

const cosMountDriverName = "ibm/ibmc-s3fs"

// TODO: Fix copy-paste from trainer/storage/s3_object_store.go, to avoid circular ref
const (
	// DataStoreTypeS3 is the type string for the S3-based object store.
	DataStoreTypeMountVolume = "mount_volume"

	// This at the level of the data or result volume
	DataStoreHostMountVolume = "host_mount"

	DataStoreTypeMountCOSS3 = "mount_cos"
	DataStoreTypeS3 = "s3_datastore"
	defaultRegion   = "us-standard"
)

//COSVolume ...
type COSVolume struct {
	VolumeType, ID, Region, Bucket, Endpoint, SecretRef, CacheSize, DiskFree, HostPath string
	MountSpec      VolumeMountSpec
}

//Volumes ...
type Volumes struct {
	TrainingData *COSVolume
	ResultsDir   *COSVolume
}

//VolumeMountSpec ...
type VolumeMountSpec struct {
	MountPath, SubPath, Name string
}

//CreateVolumeForLearner ...
func (volumes Volumes) CreateVolumeForLearner() []v1core.Volume {
	logr := logger.LocLogger(log.StandardLogger().WithField(logger.LogkeyModule, logger.LogkeyLcmService))

	var volumeSpecs []v1core.Volume

	if volumes.TrainingData != nil {
		trainingDataParams := volumes.TrainingData
		logr.Debugf("trainingDataParams.VolumeType: %s", trainingDataParams.VolumeType)
		if trainingDataParams.VolumeType == DataStoreTypeMountCOSS3 {
			volumeSpecs = append(volumeSpecs,
				generateCOSTrainingDataVolume(trainingDataParams.ID,
					trainingDataParams.Region,
					trainingDataParams.Bucket,
					trainingDataParams.Endpoint,
					trainingDataParams.SecretRef,
					trainingDataParams.CacheSize,
					trainingDataParams.DiskFree))
		} else if trainingDataParams.VolumeType == DataStoreHostMountVolume  {
			logr.Debugf("Calling generateHostMountTrainingDataVolume(%s, %s)",
				trainingDataParams.ID,
				trainingDataParams.HostPath)
			volumeSpecs = append(volumeSpecs,
				generateHostMountTrainingDataVolume(
					trainingDataParams.ID,
					trainingDataParams.HostPath))
		}
	}

	if volumes.ResultsDir != nil {
		resultDirParams := volumes.ResultsDir
		logr.Debugf("resultDirParams.VolumeType: %s", resultDirParams.VolumeType)
		if resultDirParams.VolumeType == DataStoreTypeMountCOSS3 {
			volumeSpecs = append(volumeSpecs,
				generateCOSResultsVolume(resultDirParams.ID,
					resultDirParams.Region,
					resultDirParams.Bucket,
					resultDirParams.Endpoint,
					resultDirParams.SecretRef,
					resultDirParams.CacheSize,
					resultDirParams.DiskFree))
		} else if resultDirParams.VolumeType == DataStoreHostMountVolume  {
			logr.Debugf("Calling generateHostMountTrainingDataVolume(%s, %s)",
				resultDirParams.ID,
				resultDirParams.HostPath)
			volumeSpecs = append(volumeSpecs,
				generateHostMountTrainingDataVolume(
					resultDirParams.ID,
					resultDirParams.HostPath))
		}

	}
	bytes, err := yaml.Marshal(volumeSpecs)
	if err != nil {
		logr.WithError(err).Warning("Could not marshal volumeSpecs for diagostic logging!")
	} else {
		logr.Debugf("volumeSpecs:\n %s\n", string(bytes))
	}

	return volumeSpecs
}

//CreateVolumeMountsForLearner ...
func (volumes Volumes) CreateVolumeMountsForLearner() []v1core.VolumeMount {
	logr := logger.LocLogger(log.StandardLogger().WithField(logger.LogkeyModule, logger.LogkeyLcmService))

	var mounts []v1core.VolumeMount
	if volumes.TrainingData != nil {
		mounts = append(mounts,
			generateDataDirVolumeMount(volumes.TrainingData.ID,
				volumes.TrainingData.MountSpec.MountPath,
				volumes.TrainingData.MountSpec.SubPath))
	}

	if volumes.ResultsDir != nil {
		mounts = append(mounts,
			generateResultDirVolumeMount(volumes.ResultsDir.ID,
				volumes.ResultsDir.MountSpec.MountPath,
				volumes.ResultsDir.MountSpec.SubPath))
	}

	bytes, err := yaml.Marshal(volumes)
	if err != nil {
		logr.WithError(err).Warning("Could not marshal volumes mount specs for diagostic logging!")
	} else {
		logr.Debugf("volumes mount specs:\n %s\n", string(bytes))
	}

	return mounts
}

func generateCOSTrainingDataVolume(id, region, bucket, endpoint, secretRef, cacheSize, diskfree string) v1core.Volume {
	cosInputVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			FlexVolume: &v1core.FlexVolumeSource{
				Driver:    cosMountDriverName,
				FSType:    "",
				SecretRef: &v1core.LocalObjectReference{Name: secretRef},
				ReadOnly:  true,
				Options: map[string]string{
					"bucket":           bucket,
					"endpoint":         endpoint,
					"region":           region,
					"cache-size-gb":    cacheSize, // Amount of host memory to use for cache
					"chunk-size-mb":    "52",      // value suggested for cruiser10 by benchmarking with a dallas COS instance
					"parallel-count":   "5",       // should be at least expected file size / chunk size.  Extra threads will just sit idle
					"ensure-disk-free": diskfree,  // don't completely fill the cache, leave some buffer for parallel thread pulls
					"tls-cipher-suite": "DEFAULT",
					"multireq-max":     "20",
					"stat-cache-size":  "100000",
					"kernel-cache":     "true",
					"debug-level":      "warn",
					"curl-debug":       "false",
				},
			},
		},
	}

	return cosInputVolume
}

func generateHostMountTrainingDataVolume(id, path string) v1core.Volume {
	cosInputVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			HostPath: &v1core.HostPathVolumeSource{
				Path: path,
			},
		},
	}

	return cosInputVolume
}


func generateCOSResultsVolume(id, region, bucket, endpoint, secretRef, cacheSize, diskfree string) v1core.Volume {
	cosOutputVolume := v1core.Volume{
		Name: id,
		VolumeSource: v1core.VolumeSource{
			FlexVolume: &v1core.FlexVolumeSource{
				Driver:    cosMountDriverName,
				FSType:    "",
				SecretRef: &v1core.LocalObjectReference{Name: secretRef},
				ReadOnly:  false,
				Options: map[string]string{
					"bucket":   bucket,
					"endpoint": endpoint,
					"region":   region,
					// tuning values suitable for writing checkpoints and logs
					"cache-size-gb":    cacheSize,
					"chunk-size-mb":    "52",
					"parallel-count":   "2",
					"ensure-disk-free": diskfree,
					"tls-cipher-suite": "DEFAULT",
					"multireq-max":     "20",
					"stat-cache-size":  "100000",
					"kernel-cache":     "false",
					"debug-level":      "warn",
					"curl-debug":       "false",
				},
			},
		},
	}

	return cosOutputVolume
}

func generateDataDirVolumeMount(id, bucket string, subPath string) v1core.VolumeMount {
	return v1core.VolumeMount{
		Name:      id,
		MountPath: bucket,
		SubPath: subPath,
	}
}

func generateResultDirVolumeMount(id, bucket string, subPath string) v1core.VolumeMount {
	return v1core.VolumeMount{
		Name:      id,
		MountPath: bucket,
		SubPath: subPath,
	}
}
