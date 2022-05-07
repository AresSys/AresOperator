package tensorflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ fcommon.Framework = &TensorflowFramework{}

var TensorflowBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &TensorflowFramework{Reconciler: r}
}

type TensorflowFramework struct {
	*reconciler.Reconciler
}

/*
	tensorflow框架基本参照kubeflow的实现：
	https://github.com/kubeflow/tf-operator
*/

func getSpecificRoles() []commonv1.ReplicaType {
	return []commonv1.ReplicaType{
		aresv1.RoleChief,
		aresv1.RoleEvaluator,
		aresv1.RoleMaster,
		aresv1.RolePS,
		aresv1.RoleWorker,
	}
}

func (f *TensorflowFramework) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkTensorflow
}

func (f *TensorflowFramework) GetDefaultContainerName() string {
	return f.GetName().String()
}

func (f *TensorflowFramework) GetDefaultContainerPortName() string {
	return DefaultPortName
}

func (f *TensorflowFramework) SetDefault(job *aresv1.AresJob) {
	// set framework spec
	if job.Spec.Framework.Tensorflow == nil {
		job.Spec.Framework.Tensorflow = &aresv1.TensorflowSpec{
			EnableDynamicWorker: false,
		}
	}

	specificRoles := getSpecificRoles()
	existentSpecificRoles := aresv1.GetRolesIntersection(job.Spec.RoleSpecs, specificRoles)
	hasChief := job.Spec.HasRoles(aresv1.RoleChief)
	// jobPhasePolicy
	// 默认策略为：如果没有chief，任一角色失败则任务失败；如果有chief，worker失败被拉起
	if job.Spec.JobPhasePolicy == nil {
		job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{}
		succeedAcquired := aresv1.GetRolesIntersection(job.Spec.RoleSpecs, []commonv1.ReplicaType{aresv1.RoleMaster, aresv1.RoleChief, aresv1.RoleWorker})
		if len(succeedAcquired) != 0 {
			job.Spec.JobPhasePolicy.Succeeded = [][]commonv1.ReplicaType{succeedAcquired}
		}

		for _, r := range existentSpecificRoles {
			if hasChief && r == aresv1.RoleWorker {
				continue
			}
			job.Spec.JobPhasePolicy.Failed = append(job.Spec.JobPhasePolicy.Failed, []commonv1.ReplicaType{r})
		}
	}

	// rolePhasePolicy
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	for _, role := range specificRoles {
		if spec, ok := job.Spec.RoleSpecs[role]; ok {
			if spec.RolePhasePolicy.Succeeded == nil {
				spec.RolePhasePolicy.Succeeded = &all
			}
			//有chief角色时，worker角色可以被拉起
			if hasChief && role == aresv1.RoleWorker {
				if spec.RestartPolicy != commonv1.RestartPolicyAlways {
					spec.RestartPolicy = commonv1.RestartPolicyOnFailure
				}
			}
			if spec.RestartPolicy == commonv1.RestartPolicyNever ||
				spec.RestartPolicy == commonv1.RestartPolicyExitCode {
				if spec.RolePhasePolicy.Failed == nil {
					spec.RolePhasePolicy.Failed = &any
				}
			}
		}
	}
	// set default port
	for _, role := range specificRoles {
		if spec, ok := job.Spec.RoleSpecs[role]; ok {
			f.setDefaultPort(&spec.Template.Spec)
		}
	}
}

// setDefaultPort sets the default ports for tensorflow container.
func (f *TensorflowFramework) setDefaultPort(spec *corev1.PodSpec) {
	hasDefaultContainer := false
	index := 0
	for i, container := range spec.Containers {
		if container.Name == f.GetDefaultContainerName() {
			hasDefaultContainer = true
			index = i
			break
		}
	}

	if !hasDefaultContainer {
		return
	}

	hasTFJobPort := false
	for _, port := range spec.Containers[index].Ports {
		if port.Name == DefaultPortName {
			hasTFJobPort = true
			break
		}
	}
	if !hasTFJobPort {
		spec.Containers[index].Ports = append(spec.Containers[index].Ports, v1.ContainerPort{
			Name:          DefaultPortName,
			ContainerPort: DefaultPort,
		})
	}
}

func (f *TensorflowFramework) HasDefaultContainer(spec *corev1.PodSpec) bool {
	for _, container := range spec.Containers {
		if container.Name == f.GetDefaultContainerName() {
			return true
		}
	}
	return false
}

func (f *TensorflowFramework) Validate(job *aresv1.AresJob) error {
	specificRoles := getSpecificRoles()
	for _, role := range specificRoles {
		spec, ok := job.Spec.RoleSpecs[role]
		if !ok {
			continue
		}
		if !f.HasDefaultContainer(&spec.Template.Spec) {
			return fmt.Errorf("role(%v) must have container named %v", role, f.GetDefaultContainerName())
		}
	}

	for _, role := range []commonv1.ReplicaType{aresv1.RoleMaster, aresv1.RoleChief} {
		if spec, ok := job.Spec.RoleSpecs[role]; ok {
			if spec.Replicas == nil {
				continue
			}
			if *spec.Replicas > 1 {
				return fmt.Errorf("replicas of %v cannot be greater than 1", role)
			}
		}
	}

	return nil
}

func (f *TensorflowFramework) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	aresJob := job.(*aresv1.AresJob)

	// Do not set TF_CONFIG for local training jobs.
	if !isDistributed(aresJob) {
		return nil
	}

	// Generate TF_CONFIG JSON string.
	tfConfigStr, err := f.genTFConfigJSONStr(aresJob, rtype, index)
	if err != nil {
		return err
	}

	if tfConfigStr == "" {
		return nil
	}

	// Add TF_CONFIG environment variable to tensorflow container in the pod.
	for i := range podTemplate.Spec.Containers {
		if podTemplate.Spec.Containers[i].Name == f.GetDefaultContainerName() {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, v1.EnvVar{
				Name:  tfConfig,
				Value: tfConfigStr,
			})
			break
		}
	}

	return nil

}

// isDistributed returns if the TFJob is a distributed training job.
// Ref https://github.com/kubeflow/tf-operator/issues/1078.
func isDistributed(job *aresv1.AresJob) bool {
	distributionCount := job.Spec.GetRolesReplicas(getSpecificRoles()...)
	return distributionCount > 1
}

// genClusterSpec will generate ClusterSpec.
func (f *TensorflowFramework) genClusterSpec(job *aresv1.AresJob) (ClusterSpec, error) {
	clusterSpec := make(ClusterSpec)

	specificRoles := getSpecificRoles()
	for _, role := range specificRoles {
		_, ok := job.Spec.RoleSpecs[role]
		if !ok {
			continue
		}

		endPoints := []string{}

		port, err := f.GetPort(job, role)
		if err != nil {
			return nil, err
		}

		roleDomainNames := reconciler.GetRolesDomainNames(job, role)
		for _, dn := range roleDomainNames {
			endPoint := fmt.Sprintf("%s:%d", dn, port)
			endPoints = append(endPoints, endPoint)
		}

		clusterSpec[string(role)] = endPoints
	}

	return clusterSpec, nil
}

// genTFConfig will generate the environment variable TF_CONFIG
// {
//     "cluster": {
//         "ps": ["ps1:2222", "ps2:2222"],
//         "worker": ["worker1:2222", "worker2:2222", "worker3:2222"]
//     },
//     "task": {
//         "type": "ps",
//         "index": 1
//         },
//     }
// }
func (f *TensorflowFramework) genTFConfigJSONStr(job *aresv1.AresJob, rtype, index string) (string, error) {
	// Configure the TFCONFIG environment variable.
	i, err := strconv.ParseInt(index, 0, 32)
	if err != nil {
		return "", err
	}

	cluster, err := f.genClusterSpec(job)
	if err != nil {
		return "", err
	}

	var tfConfigJSONByteSlice []byte
	if enableDynamicWorker(job) {
		sparseCluster := convertClusterSpecToSparseClusterSpec(cluster, strings.ToLower(rtype), int32(i))
		sparseTFConfig := SparseTFConfig{
			Cluster: sparseCluster,
			Task: TaskSpec{
				Type:  strings.ToLower(rtype),
				Index: int(i),
			},
		}
		tfConfigJSONByteSlice, err = json.Marshal(sparseTFConfig)
	} else {
		tfConfig := TFConfig{
			Cluster: cluster,
			Task: TaskSpec{
				Type:  strings.ToLower(rtype),
				Index: int(i),
			},
			// We need to set environment to cloud  otherwise it will default to local which isn't what we want.
			// Environment is used by tensorflow.contrib.learn.python.learn in versions <= 1.3
			// TODO(jlewi): I don't think it is used in versions TF >- 1.4. So we can eventually get rid of it.
			Environment: "cloud",
		}
		tfConfigJSONByteSlice, err = json.Marshal(tfConfig)
	}
	if err != nil {
		return "", err
	}

	return string(tfConfigJSONByteSlice), nil
}

func enableDynamicWorker(job *aresv1.AresJob) bool {
	return job.Spec.Framework.Tensorflow.EnableDynamicWorker
}
