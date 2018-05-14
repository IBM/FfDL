package learner

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/IBM/FfDL/commons/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateStatefulSetSpecForLearner(t *testing.T) {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	namespace := config.GetLearnerNamespace()
	podSpec := createPodSpecForTesting()
	statefulSetService := CreateServiceSpec("statefulset-service", "nonSplitSingleLearner-trainingID")
	statefulSet := CreateStatefulSetSpecForLearner("statefulset", statefulSetService.Name, 2, podSpec)
	clientSet := fake.NewSimpleClientset(statefulSet, statefulSetService)
	clientSet.CoreV1().Services(namespace).Create(statefulSetService)

	_, err := clientSet.AppsV1beta1().StatefulSets(namespace).Create(statefulSet)
	assert.NoError(t, err)
	_, err = clientSet.AppsV1beta1().StatefulSets(namespace).Get("statefulset", metav1.GetOptions{})

	assert.NoError(t, err)

	clientSet.CoreV1().Pods(namespace).Get("statefulset", metav1.GetOptions{})
}
