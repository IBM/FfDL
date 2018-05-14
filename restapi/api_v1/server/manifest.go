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

package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/IBM/FfDL/commons/logger"
	"github.com/IBM/FfDL/trainer/trainer/grpc_trainer_v2"

	"strconv"

	"gopkg.in/yaml.v2"
)

// ManifestV1 represents a manifest used to define the configurations for a training job
type ManifestV1 struct {
	Name              string            `yaml:"name,omitempty"`
	Description       string            `yaml:"description,omitempty"`
	Version           string            `yaml:"version,omitempty"`
	Cpus              float64           `yaml:"cpus,omitempty"`
	Gpus              float64           `yaml:"gpus,omitempty"`
	Gpu_type	  string	    `yaml:"gpu_type,omitempty"`
	Learners          int32             `yaml:"learners,omitempty"`
	Memory            string            `yaml:"memory,omitempty"`
	Storage           string            `yaml:"storage,omitempty"`
	DataStores        []*dataStoreRef   `yaml:"data_stores,omitempty"`
	Framework         *frameworkV1      `yaml:"framework,omitempty"`
	EvaluationMetrics *EMExtractionSpec `yaml:"evaluation_metrics,omitempty"`
}

// EMExtractionSpec specifies which log-collector is run, and how the evaluation metrics are extracted.
type EMExtractionSpec struct {
	Type          string              `yaml:"type,omitempty"`
	ImageTag      string              `yaml:"image_tag,omitempty"`
	In            string              `yaml:"in,omitempty"`
	LineLookahead int32               `yaml:"line_lookahead,omitempty"`
	EventTypes    []string            `yaml:"eventTypes,omitempty"`
	Groups        map[string]*EMGroup `yaml:"groups,omitempty"`
}

// EMGroup is used by the regex_extractor to specify log line matches, and the corresponding
// evaluation metrics.
type EMGroup struct {
	Regex   string          `yaml:"regex,omitempty"`
	Meta    *EMMeta         `yaml:"meta,omitempty"`
	Scalars map[string]*Any `yaml:"scalars,omitempty"`
	Etimes  map[string]*Any `yaml:"etimes,omitempty"`
}

// Any is used for typed values.
type Any struct {
	Type  string `yaml:"type,omitempty"`
	Value string `yaml:"value,omitempty"`
}

// EMMeta is used in the EMGroup record to specify the time occurrence of the evaluation metric.
type EMMeta struct {
	// Time that the metric occured: representing the number of millisecond since midnight January 1, 1970.
	// (ref, for instance $timestamp).  Value will be extracted from timestamps
	Time string `yaml:"time,omitempty"`
}

type dataStoreRef struct {
	ID              string              `yaml:"id,omitempty"`
	Type            string              `yaml:"type,omitempty"`
	TrainingData    *storageContainerV1 `yaml:"training_data,omitempty"`
	TrainingResults *storageContainerV1 `yaml:"training_results,omitempty"`
	Connection      map[string]string   `yaml:"connection,omitempty"`
}

type frameworkV1 struct {
	Name     string `yaml:"name,omitempty"`
	Version  string `yaml:"version,omitempty"`
	ImageTag string `yaml:"image_tag,omitempty"`
	Command  string `yaml:"command,omitempty"`
}

type storageContainerV1 struct {
	Container string `yaml:"container,omitempty"`
}

// LoadManifestV1 constructs a manifest object from a byte array
func LoadManifestV1(data []byte) (*ManifestV1, error) {
	t := &ManifestV1{}
	err := yaml.Unmarshal([]byte(data), &t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// WriteManifestV1 writes a manifest object to a byte array
func WriteManifestV1(t *ManifestV1) ([]byte, error) {
	data, err := yaml.Marshal(t)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func validateEvaluationMetricsSpec(m *ManifestV1) error {
	if m.EvaluationMetrics != nil {
		//	TODO: implement evaluation metrics spec validation?
	}
	return nil
}

//func typeStringToGRPCType(t string) grpc_trainer_v2.Any_DataType {
//	t2 := grpc_trainer_v2.Any_DataType_value[t]
//	return grpc_trainer_v2.Any_DataType(t2)
//}

func marshalToTrainerEvaluationMetricsSpec(em *EMExtractionSpec) *grpc_trainer_v2.EMExtractionSpec {
	groups := make(map[string]*grpc_trainer_v2.EMGroup)

	for key, emGroup := range em.Groups {
		values := make(map[string]*grpc_trainer_v2.EMAny)

		for scalerKey, typedValue := range emGroup.Scalars {
			marshaledValue :=  &grpc_trainer_v2.EMAny{Type: typedValue.Type, Value: typedValue.Value}
			values[scalerKey] = marshaledValue
		}
		etimes := make(map[string]*grpc_trainer_v2.EMAny)
		for etimeKey, typedValue := range emGroup.Etimes {
			marshaledValue :=  &grpc_trainer_v2.EMAny{Type: typedValue.Type, Value: typedValue.Value}
			etimes[etimeKey] = marshaledValue
		}
		group := &grpc_trainer_v2.EMGroup{
			Regex: emGroup.Regex,
			Meta: &grpc_trainer_v2.EMMeta{
				Time: emGroup.Meta.Time,
			},
			Values: values,
			Etimes: etimes,
		}
		groups[key] = group
	}

	return &grpc_trainer_v2.EMExtractionSpec{
		Type:          em.Type,
		ImageTag:      em.ImageTag,
		In:            em.In,
		LineLookahead: em.LineLookahead,
		EventTypes:    em.EventTypes,
		Groups:        groups,
	}
}

// manifest2TrainingRequest converts the existing manifest to a training request for the trainer microservice.
func manifest2TrainingRequest(m *ManifestV1, modelDefinition []byte, http *http.Request,
	logr *logger.LocLoggingEntry) *grpc_trainer_v2.CreateRequest {

	r := &grpc_trainer_v2.CreateRequest{
		UserId: getUserID(http),
		ModelDefinition: &grpc_trainer_v2.ModelDefinition{
			Name:        m.Name,
			Description: m.Description,
			Framework: &grpc_trainer_v2.Framework{
				Name:     m.Framework.Name,
				Version:  m.Framework.Version,
				ImageTag: m.Framework.ImageTag,
			},
			Content: modelDefinition,
		},
		Training: &grpc_trainer_v2.Training{
			Command:   m.Framework.Command,
			InputData: []string{m.DataStores[0].ID + "-input"},
			Profiling: false,
		},
		Datastores: []*grpc_trainer_v2.Datastore{
			{
				Id:         m.DataStores[0].ID + "-input",
				Type:       m.DataStores[0].Type,
				Connection: m.DataStores[0].Connection,
				Fields: map[string]string{
					"bucket": m.DataStores[0].TrainingData.Container,
				},
			},
		},
	}

	if m.DataStores[0].TrainingResults != nil {
		r.Training.OutputData = []string{m.DataStores[0].ID + "-output"}
		r.Datastores = append(r.Datastores, &grpc_trainer_v2.Datastore{
			Id:         m.DataStores[0].ID + "-output",
			Type:       m.DataStores[0].Type,
			Connection: m.DataStores[0].Connection,
			Fields: map[string]string{
				"bucket": m.DataStores[0].TrainingResults.Container,
			},
		})
	}

	mem, memUnit, err := convertMemoryFromManifest(m.Memory)
	if err != nil {
		logr.WithError(err).Errorf("Incorrect memory specification in manifest")
	}

	storage, storageUnit := float32(0), grpc_trainer_v2.SizeUnit_MB // default to 0
	if m.Storage != "" {
		storage, storageUnit, err = convertMemoryFromManifest(m.Storage)
		if err != nil {
			logr.WithError(err).Errorf("Incorrect storage specification in manifest")
		}
	}

	r.Training.Resources = &grpc_trainer_v2.ResourceRequirements{
		Gpus:        float32(m.Gpus),
		GpuType:     string(m.Gpu_type),
		Cpus:        float32(m.Cpus),
		Memory:      mem,
		MemoryUnit:  memUnit,
		Storage:     storage,
		StorageUnit: storageUnit,
		Learners:    m.Learners,
		// TODO add storage support
	}

	if m.EvaluationMetrics != nil {
		err = validateEvaluationMetricsSpec(m)
		if err != nil {
			logr.WithError(err).Errorf("Incorrect evaluation metrics specification in manifest")
		}

		r.EvaluationMetrics = marshalToTrainerEvaluationMetricsSpec(m.EvaluationMetrics)
		logr.Debugf("EMExtractionSpec ImageTag: %s", r.EvaluationMetrics.ImageTag)
	}

	return r
}

// Converts an incoming memory spec like 4GB or 1024mb into a memory (float64)
// and a unit.
// Rules for conversion:
// 1GB (gigabytes) = 1000MB (megabytes)
// 1Gi (gibibytes) = 1024 Mi (mebibytes)
func convertMemoryFromManifest(memStr string) (float32, grpc_trainer_v2.SizeUnit, error) {
	regex, _ := regexp.Compile(`(?i)(?P<memory>(\d+)?)\s*(?P<unit>(mb|mib|gb|gib|tb|tib)?)`)
	match := regex.FindStringSubmatch(memStr)

	// convert to a map
	result := make(map[string]string)
	for i, name := range regex.SubexpNames() {
		if i != 0 && i <= len(match) {
			result[name] = match[i]
		}
	}
	mem, err := strconv.ParseFloat(result["memory"], 64)
	if err != nil {
		return 512, grpc_trainer_v2.SizeUnit_MB, fmt.Errorf("Memory requirements not correctly specified: %s", memStr)
	}
	unit := strings.ToLower(result["unit"])

	for k, v := range grpc_trainer_v2.SizeUnit_value {
		if strings.ToLower(k) == unit {
			return float32(mem), grpc_trainer_v2.SizeUnit(v), nil
		}
	}
	return 512, grpc_trainer_v2.SizeUnit_MB, fmt.Errorf("Memory requirements not correctly specified: %s", memStr)
}
