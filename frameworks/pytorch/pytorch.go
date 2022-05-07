package pytorch

import (
	"fmt"
	"strconv"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
)

var _ fcommon.Framework = &PytorchFramework{}

var PytorchBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &PytorchFramework{Reconciler: r}
}

type PytorchFramework struct {
	*reconciler.Reconciler
}

func (f *PytorchFramework) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkPytorch
}

func (f *PytorchFramework) GetDefaultContainerName() string {
	return f.GetName().String()
}

func (f *PytorchFramework) GetDefaultContainerPortName() string {
	return f.GetName().String()
}

func (f *PytorchFramework) SetDefault(job *aresv1.AresJob) {
	// jobPhasePolicy
	if job.Spec.JobPhasePolicy == nil {
		policy := &aresv1.JobPhasePolicy{
			Failed:    [][]commonv1.ReplicaType{{aresv1.RoleMaster}},
			Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleMaster}},
		}
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
			policy = &aresv1.JobPhasePolicy{
				Failed:    [][]commonv1.ReplicaType{{aresv1.RoleMaster}, {aresv1.RoleWorker}},
				Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleMaster, aresv1.RoleWorker}},
			}
		}
		job.Spec.JobPhasePolicy = policy
	}
	// rolePhasePolicy
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
		}
		if spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
	}
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleMaster]; ok {
		if spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
		}
		if spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
	}
	// dependence
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleMaster]; ok && spec.Dependence == nil {
			spec.Dependence = &aresv1.DependenceSpec{
				RoleNames: []commonv1.ReplicaType{aresv1.RoleMaster},
				RolePhase: aresv1.RolePhaseRunning,
			}
		}
	}
}

func (f *PytorchFramework) Validate(job *aresv1.AresJob) error {
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleMaster]; !ok {
		return fmt.Errorf("role %s is required for %s: %s", aresv1.RoleMaster, f.GetName(), job.Name)
	} else if *spec.Replicas != 1 {
		return fmt.Errorf("replicas for role %s must be 1: %d", aresv1.RoleMaster, *spec.Replicas)
	}
	return nil
}

func (f *PytorchFramework) SetClusterSpec(j interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	if rtype != string(aresv1.RoleMaster) && rtype != string(aresv1.RoleWorker) {
		return nil
	}
	// rank
	rank, err := strconv.Atoi(index)
	if err != nil {
		return fmt.Errorf("failed to convert string(%s) to int: %v", index, err)
	}
	// master address
	job := j.(*aresv1.AresJob)
	masterAddr := "0.0.0.0"
	if rtype == string(aresv1.RoleWorker) {
		masterAddr = reconciler.GetPodDomainName(job.Name, job.Namespace, aresv1.RoleMaster, 0)
		rank = rank + 1
	}
	// master port
	masterPort, err := f.GetPort(job, aresv1.RoleMaster)
	if err != nil {
		return fmt.Errorf("failed to get pytorch master port: %v", err)
	}

	// add env
	for i := range podTemplate.Spec.Containers {
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, []corev1.EnvVar{
			{Name: "MASTER_ADDR", Value: masterAddr},
			{Name: "MASTER_PORT", Value: strconv.Itoa(int(masterPort))},
			{Name: "WORLD_SIZE", Value: strconv.Itoa(int(job.Spec.GetRoleReplicas(aresv1.RoleWorker)) + 1)},
			{Name: "RANK", Value: strconv.Itoa(rank)},
		}...)
	}
	return nil
}

// ReconcileServices: pytorch框架无需对worker创建service
func (f *PytorchFramework) ReconcileServices(
	job metav1.Object,
	services []*corev1.Service,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {

	log := util.LoggerForJob(job)
	if rtype == aresv1.RoleWorker {
		log.Debugf("I don't need service: %s/%s", f.GetName(), rtype)
		return nil
	}

	return f.Reconciler.ReconcileServices(job, services, rtype, spec)
}
