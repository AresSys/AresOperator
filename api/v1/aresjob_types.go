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

import (
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

/*********
 * Spec
 *********/
// AresJobSpec defines the desired state of AresJob
type AresJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	TTLSecondsAfterFailed *int32                             `json:"ttlSecondsAfterFailed,omitempty"`
	RunPolicy             commonv1.RunPolicy                 `json:"runPolicy,omitempty"`
	AvailablePorts        []int32                            `json:"availablePorts,omitempty"`
	FrameworkType         FrameworkType                      `json:"frameworkType,omitempty"`
	Framework             FrameworkSpec                      `json:"framework,omitempty"`
	RoleSpecs             map[commonv1.ReplicaType]*RoleSpec `json:"roleSpecs,omitempty"`
	JobPhasePolicy        *JobPhasePolicy                    `json:"jobPhasePolicy,omitempty"`
}

// GetReplicaSpecs: get replicaSpecs
func (spec AresJobSpec) GetReplicaSpecs() map[commonv1.ReplicaType]*commonv1.ReplicaSpec {
	replicaSpecs := map[commonv1.ReplicaType]*commonv1.ReplicaSpec{}
	for replicaType, roleSpec := range spec.RoleSpecs {
		replicaSpecs[replicaType] = roleSpec.ReplicaSpec
	}
	return replicaSpecs
}

func (s AresJobSpec) GetTotalReplicas() int32 {
	count := int32(0)
	for _, spec := range s.RoleSpecs {
		if spec != nil && spec.Replicas != nil {
			count += *spec.Replicas
		}
	}
	return count
}

func (s AresJobSpec) GetRolesReplicas(roles ...commonv1.ReplicaType) int32 {
	count := int32(0)
	for _, role := range roles {
		count += s.GetRoleReplicas(role)
	}
	return count
}

func (s AresJobSpec) GetRoleReplicas(role commonv1.ReplicaType) int32 {
	if s.RoleSpecs[role] != nil && s.RoleSpecs[role].Replicas != nil {
		return *s.RoleSpecs[role].Replicas
	}
	return 0
}

func (s AresJobSpec) HasRoles(roles ...commonv1.ReplicaType) bool {
	for _, role := range roles {
		if _, ok := s.RoleSpecs[role]; !ok {
			return false
		}
	}
	return true
}

// EnableGangScheduling: 是否开启GangScheduling
func (spec AresJobSpec) EnableGangScheduling() bool {
	return spec.RunPolicy.SchedulingPolicy != nil && spec.RunPolicy.SchedulingPolicy.MinAvailable != nil
}

// HasDependence: 是否包含依赖
func (spec AresJobSpec) HasDependence() bool {
	for _, roleSpec := range spec.RoleSpecs {
		if roleSpec.Dependence != nil {
			return true
		}
	}
	return false
}

func (spec AresJobSpec) CheckRolesExist(roles []commonv1.ReplicaType) error {
	for _, role := range roles {
		info, ok := spec.RoleSpecs[role]
		if !ok || info == nil {
			return fmt.Errorf("roles(%v) not exist", roles)
		}
	}

	return nil
}

// RoleSpec: 角色通用字段
type RoleSpec struct {
	*commonv1.ReplicaSpec `json:",inline"`
	RolePhasePolicy       RolePhasePolicy `json:"rolePhasePolicy,omitempty"`
	Controller            ControllerType  `json:"controller,omitempty"`
	Dependence            *DependenceSpec `json:"dependence,omitempty"`
}

// DependenceSpec: 角色依赖关系
type DependenceSpec struct {
	RoleNames   []commonv1.ReplicaType `json:"roleNames,omitempty"`
	RolePhase   RolePhase              `json:"rolePhase,omitempty"`
	MPIHostFile *string                `json:"mpiHostFile,omitempty"`
}

/*********
 * Status
 *********/
// AresJobStatus defines the observed state of AresJob
type AresJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	*commonv1.JobStatus `json:",inline"`
	RoleStatuses        map[commonv1.ReplicaType]*RoleStatus `json:"roleStatuses,omitempty"`
	Phase               JobPhase                             `json:"phase,omitempty"`
	RunningTime         *metav1.Time                         `json:"runningTime,omitempty"`
}

func (status AresJobStatus) RolePhase(rtype commonv1.ReplicaType) RolePhase {
	if s, ok := status.RoleStatuses[rtype]; ok {
		return s.Phase
	}
	return RolePhasePending
}

// Evicted: if job is evicted
func (status AresJobStatus) Evicted() bool {
	if status.Phase != JobPhaseFailed {
		return false
	}
	for _, cond := range status.Conditions {
		if cond.Type == commonv1.JobFailed &&
			cond.Status == corev1.ConditionTrue &&
			cond.Reason == Preempted {
			return true
		}
	}
	return false
}

// RoleStatus: 角色状态
type RoleStatus struct {
	Phase              RolePhase   `json:"phase,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	ReplicaStatus      `json:",inline"`
}

// ReplicaStatus: represents the current observed state of the replica.
type ReplicaStatus struct {
	*commonv1.ReplicaStatus `json:",inline"`
	Pending                 int32 `json:"pending,omitempty"`
	Unknown                 int32 `json:"unknown,omitempty"`
	Terminating             int32 `json:"terminating,omitempty"`
}

/*********
 * AresJob
 *********/
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=aj
//+kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.metadata.namespace`
//+kubebuilder:printcolumn:name="Framework",type=string,JSONPath=`.spec.frameworkType`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AresJob is the Schema for the aresjobs API
type AresJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AresJobSpec   `json:"spec,omitempty"`
	Status AresJobStatus `json:"status,omitempty"`
}

func (job AresJob) Json() string {
	if content, err := json.Marshal(job); err == nil {
		return string(content)
	}
	return fmt.Sprintf("AresJob{f=%s, %s.%s}", job.Spec.FrameworkType, job.Namespace, job.Name)
}

//+kubebuilder:object:root=true

// AresJobList contains a list of AresJob
type AresJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AresJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AresJob{}, &AresJobList{})
}
