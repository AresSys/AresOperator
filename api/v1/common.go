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
	"errors"
	"fmt"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

const (
	// gang scheduler name.
	GangSchedulerName = "volcano"
	// GangSchedulingPodGroupAnnotation is the annotation key used by batch schedulers
	GangSchedulingPodGroupAnnotation = "scheduling.k8s.io/group-name"
	// MPIHostFilePath is the default path for mpi host file
	DefaultMPIHostFilePath = "/etc/mpi/hostfile"
	// name of Aresjob finalizer
	AresJobFinalizer = "ares.io/ares-finalizer"
)

var (
	JobOwnerKey     = ".metadata.controller"
	ReplicaOwnerKey = ".metadata.replica-type"
	NodeNameKey     = ".spec.nodeName"
	ErrPortNotFound = errors.New("container port was not found")
)

func MetaReplicaKeyFormat(jobName string, rtype commonv1.ReplicaType) string {
	return fmt.Sprintf("%s/%s", jobName, strings.ToLower(string(rtype)))
}

/************
 * Controller
 ************/
type ControllerType string

const (
	ControllerPod         ControllerType = "Pod"
	ControllerDeployment  ControllerType = "Deployment"
	ControllerStatefulSet ControllerType = "StatefulSet"
	ControllerJob         ControllerType = "Job"
)

/************
 * Condition
 ************/
const (
	JobScheduled commonv1.JobConditionType = "Scheduled"

	// reason
	Evicted   string = "Evicted"
	Preempted string = "Preempted"
)

/************
 * Framework
 ************/
type FrameworkType string

func (f FrameworkType) String() string {
	return string(f)
}

const (
	FrameworkBagua      FrameworkType = "bagua"
	FrameworkCustom     FrameworkType = "custom"
	FrameworkDask       FrameworkType = "dask"
	FrameworkDemo       FrameworkType = "demo"
	FrameworkMPI        FrameworkType = "mpi"
	FrameworkPytorch    FrameworkType = "pytorch"
	FrameworkStandalone FrameworkType = "standalone"
	FrameworkTensorflow FrameworkType = "tensorflow"
)

/************
 * Role
 ************/
const (
	RoleLauncher   commonv1.ReplicaType = "launcher"
	RoleWorker     commonv1.ReplicaType = "worker"
	RolePS         commonv1.ReplicaType = "ps"
	RoleHub        commonv1.ReplicaType = "hub"
	RoleTrain      commonv1.ReplicaType = "train"
	RoleEtcd       commonv1.ReplicaType = "etcd"
	RoleMaster     commonv1.ReplicaType = "master"
	RoleScheduler  commonv1.ReplicaType = "scheduler"
	RoleLeadWorker commonv1.ReplicaType = "leadworker"
	RoleChief      commonv1.ReplicaType = "chief"
	RoleEvaluator  commonv1.ReplicaType = "evaluator"
)
