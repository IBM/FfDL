package lcm

import (
	"github.com/IBM/FfDL/commons/config"
	"github.com/IBM/FfDL/lcm/service/lcm/helper"
	"github.com/IBM/FfDL/lcm/service/lcm/learner"
	"k8s.io/api/apps/v1beta1"
	v1core "k8s.io/api/core/v1"
)

func (t splitTraining) Start() error {

	serviceSpec := learner.CreateServiceSpec(t.learner.name, t.req.TrainingId)

	numLearners := int(t.req.GetResources().Learners)

	return t.CreateFromBOM(&splitTrainingBOM{
		t.learner.secrets,
		serviceSpec,
		t.helper.sharedVolumeClaim,
		t.statefulSetSpecForLearner(serviceSpec.Name),
		t.deploymentSpecForHelper(),
		numLearners,
	})
}

///-------
func (t splitTraining) deploymentSpecForHelper() *v1beta1.Deployment {

	helperDefn := t.helper
	helperContainers := t.constructAuxillaryContainers()

	podSpec := helper.CreatePodSpec(helperContainers, []v1core.Volume{helperDefn.etcdVolume, helperDefn.sharedVolume}, map[string]string{"training_id": t.req.TrainingId, "user_id": t.req.UserId})
	deploymentSpec := helper.CreateDeploymentForHelper(helperDefn.name, podSpec)
	return deploymentSpec

}

func (t splitTraining) statefulSetSpecForLearner(serviceName string) *v1beta1.StatefulSet {

	gpus := make(map[string]string)
	if t.req.Resources.Gpus > 0 {
		gpus["ibm-cloud.kubernetes.io/gpu-type"] = t.req.Resources.GpuType
	}

	learnerDefn := t.learner
	helperDefn := t.helper

	helperAndLearnerVolumes := append(learnerDefn.volumes, helperDefn.sharedVolume)

	//now create the learner container
	learnerContainer := constructLearnerContainer(t.req, learnerDefn.envVars, learnerDefn.volumeMounts, helperDefn.sharedVolumeMount, learnerDefn.mountTrainingDataStoreInLearner, learnerDefn.mountResultsStoreInLearner, t.logr) // nil for mounting shared NFS volume since non split mode
	nonSplitLearnerPodSpec := learner.CreatePodSpec([]v1core.Container{learnerContainer}, helperAndLearnerVolumes, map[string]string{"training_id": t.req.TrainingId, "user_id": t.req.UserId}, gpus)
	statefulSetSpec := learner.CreateStatefulSetSpecForLearner(learnerDefn.name, serviceName, learnerDefn.numberOfLearners, nonSplitLearnerPodSpec)

	return statefulSetSpec
}

//func (t *splitTraining) dumpAsYaml(label string, in interface{}) {
//	logr := t.logr
//	yamlbytes,err := yaml.Marshal(in)
//	if err != nil {
//		logr.WithError(err).Errorf("Could not marshal volume mounts for diagnosis")
//	}
//	fmt.Printf("--------------------------------------------\n")
//	fmt.Printf("------------ Full Learner Spec -------------\n")
//	fmt.Printf("%s:\n %s", label, string(yamlbytes))
//	fmt.Printf("--------------------------------------------\n")
//
//}

//CreateFromBOM ... eventually use with controller and make this transactional
func (t *splitTraining) CreateFromBOM(bom *splitTrainingBOM) error {
	logr := t.logr
	namespace := config.GetLearnerNamespace()

	//create shared volume
	if bom.sharedVolumeClaimBOM != nil { //if nil then must be static volume claim and does not need to be dynamically bound
		logr.Infof("Split training with shared volume claim %s not nil, creating shared PVC for training", bom.sharedVolumeClaimBOM.Name)

		//t.dumpAsYaml("sharedVolumeClaimBOM", bom.sharedVolumeClaimBOM)

		if err := helper.CreatePVCFromBOM(bom.sharedVolumeClaimBOM, t.k8sClient); err != nil {
			logr.WithError(err).Errorf("Failed in creating shared volume claim %s while deploying for training ", bom.sharedVolumeClaimBOM.Name)
			return err
		}
	}

	//create helper
	//t.dumpAsYaml("Deployment", bom.helperBOM)
	if _, err := t.k8sClient.AppsV1beta1().Deployments(namespace).Create(bom.helperBOM); err != nil {
		logr.WithError(err).Errorf("Failed in creating helper %s while deploying for training", bom.helperBOM.Name)
		return err
	}

	for _, secret := range bom.secrets {
		//create the secrets
		if _, err := t.k8sClient.CoreV1().Secrets(namespace).Create(secret); err != nil {
			logr.WithError(err).Errorf("Failed in creating secret %s while deploying for training ", secret.Name)
			return err
		}
		logr.Infof("Created secret %s", secret.Name)
	}

	if bom.numLearners > 1 {
		//create service
		if _, err := t.k8sClient.CoreV1().Services(namespace).Create(bom.service); err != nil {
			logr.WithError(err).Errorf("failed in creating services %s while deploying for training ", bom.service.Name)
			return err
		}
	}

	//create the stateful set
	if _, err := t.k8sClient.AppsV1beta1().StatefulSets(namespace).Create(bom.learnerBOM); err != nil {
		logr.WithError(err).Errorf("failed in creating statefulset %s while deploying for training ", bom.learnerBOM.Name)
		return err
	}
	return nil
}
