package standalone

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
)

var _ fcommon.Framework = &StandaloneFramework{}

var StandaloneBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &StandaloneFramework{Reconciler: r}
}

type StandaloneFramework struct {
	*reconciler.Reconciler
}

func (f *StandaloneFramework) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkStandalone
}

func (f *StandaloneFramework) GetDefaultContainerName() string {
	return f.GetName().String()
}

func (f *StandaloneFramework) SetDefault(job *aresv1.AresJob) {
	// jobPhasePolicy
	if job.Spec.JobPhasePolicy == nil {
		job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
			Failed:    [][]commonv1.ReplicaType{{aresv1.RoleWorker}},
			Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleWorker}},
		}
	}
	// rolePhasePolicy
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
		if spec.RestartPolicy == commonv1.RestartPolicyNever ||
			spec.RestartPolicy == commonv1.RestartPolicyExitCode {
			if spec.RolePhasePolicy.Failed == nil {
				spec.RolePhasePolicy.Failed = &any
			}
		}
	}
}

func (f *StandaloneFramework) Validate(job *aresv1.AresJob) error {
	return nil
}

// ReconcileServices: 无需创建service
func (f *StandaloneFramework) ReconcileServices(
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
