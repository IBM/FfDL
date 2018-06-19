/*
 * Copyright 2017-2018 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
