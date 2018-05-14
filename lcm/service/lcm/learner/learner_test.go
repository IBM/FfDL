package learner

/*
func TestCreateLearnerFromBOMForSingleLearnerNonSplitModeNonCOS(t *testing.T) {

	mountTrainingDataStoreInLearner, mountResultsStoreInLearner, mountSSHCertsInLearner := false, false, false
	trainingID := "TestCreateLearnerFromBOMForSingleLearnerNonSplitModeNonCOS"
	//create pod, service, statefuleset spec
	nonSplitLearnerPodSpec := CreatePodSpec("podname", trainingID, helperContainers, learnerVolumes)
	serviceSpec := CreateServiceSpec("service", trainingID)
	statefulSetSpec := CreateStatefulSetSpecForLearner("statefulset", serviceSpec.Name, 1, nonSplitLearnerPodSpec)

	//now wrap it with stateful set
	dependencies := StatefulSetDependencies{
		Secrets: createSecretsForTest(),
		Volumes: helperAndLearnerVolumes,
		Service: serviceSpec,
	}

	CreateLearnerFromBOM(statefulSetSpec, dependencies)

}

func createSecretsForTest() []*v1core.Secret {
	mockSecrets := Secrets{
		ResultsDirSecret:   &COSVolumeSecret{},
		TrainingDataSecret: &COSVolumeSecret{},
		SSHVolumeSecret:    &SSHVolumeSecret{},
	}
	return CreateVolumeSecretsSpec(mockSecrets)

}
*/
