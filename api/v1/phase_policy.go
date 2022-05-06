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
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

/*********
 * Job
 *********/
type JobPhase string

func (phase JobPhase) String() string {
	return string(phase)
}

const (
	JobPhasePending  JobPhase = "Pending"
	JobPhaseStarting JobPhase = "Starting"
	// JobPhaseRunning: all roles are at running
	JobPhaseRunning   JobPhase = "Running"
	JobPhaseFailed    JobPhase = "Failed"
	JobPhaseSucceeded JobPhase = "Succeeded"
)

// JobPhaseToCondition: JobPhase -> commonv1.JobConditionType
func JobPhaseToCondition(phase JobPhase) commonv1.JobConditionType {
	switch phase {
	case JobPhasePending, JobPhaseStarting:
		return commonv1.JobCreated
	case JobPhaseRunning:
		return commonv1.JobRunning
	case JobPhaseFailed:
		return commonv1.JobFailed
	case JobPhaseSucceeded:
		return commonv1.JobSucceeded
	}
	return commonv1.JobCreated
}

const (
	ErrInvalidDefinition = "InvalidDefinition"
)

/* 任务状态策略
   实例：
   Succeeded: # OR: 列表中任一条件成功，则job成功
    - [Chief, Worker] # AND: Cheif和Worker角色都Succeeded时Job成功
    - [Launcher] # Launcher角色Succeeded时Job成功
*/
type JobPhasePolicy struct {
	Failed    [][]commonv1.ReplicaType `json:"failed,omitempty"`
	Succeeded [][]commonv1.ReplicaType `json:"succeeded,omitempty"`
}

/*********
 * Role
 *********/
type RolePhase string

const (
	RolePhasePending   RolePhase = "Pending"
	RolePhaseStarting  RolePhase = "Starting"
	RolePhaseRunning   RolePhase = "Running"
	RolePhaseFailed    RolePhase = "Failed"
	RolePhaseSucceeded RolePhase = "Succeeded"
)

type RolePhasePolicy struct {
	Failed           *RolePhaseCondition `json:"failed,omitempty"`
	Succeeded        *RolePhaseCondition `json:"succeeded,omitempty"`
	NodeDownAsFailed bool                `json:"nodeDownAsFailed,omitempty"`
}

type RolePhaseCondition string

const (
	RolePhaseConditionAnyPod  RolePhaseCondition = "AnyPod"
	RolePhaseConditionAllPods RolePhaseCondition = "AllPods"
)
