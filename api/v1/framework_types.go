/*
Copyright 2021 KML Ares-Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import "fmt"

// FrameworkSpec 框架相关字段
type FrameworkSpec struct {
	Bagua      *BaguaSpec      `json:"bagua,omitempty"`
	MPI        *MPISpec        `json:"mpi,omitempty"`
	Tensorflow *TensorflowSpec `json:"tensorflow,omitempty"`
}

type BaguaSpec struct {
	EnableElastic bool   `json:"enableElastic,omitempty"`
	MinReplicas   *int32 `json:"minReplicas,omitempty"`
	MaxReplicas   *int32 `json:"maxReplicas,omitempty"`
}

type MPISpec struct {
	ReplicasPerNode int               `json:"replicasPerNode,omitempty"`
	Implementation  MPIImplementation `json:"implementation,omitempty"`
}

type MPIImplementation string

const (
	OMPI  MPIImplementation = "ompi"
	MPICH MPIImplementation = "mpich"
)

func (spec *MPISpec) Validate() error {
	if spec.Implementation != OMPI && spec.Implementation != MPICH {
		return fmt.Errorf("unsupported mpi implementation:%v", spec.Implementation)
	}
	return nil
}

// from https://github.com/kubeflow/tf-operator/blob/master/pkg/apis/tensorflow/v1/types.go#L67
type TensorflowSpec struct {
	// A switch to enable dynamic worker
	EnableDynamicWorker bool `json:"enableDynamicWorker,omitempty"`
}
