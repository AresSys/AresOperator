package mpi

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

var _ fcommon.Framework = &MPIFramework{}

var MPIBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &MPIFramework{Reconciler: r}
}

type MPIFramework struct {
	*reconciler.Reconciler
}

func (f *MPIFramework) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkMPI
}

func (f *MPIFramework) GetDefaultContainerName() string {
	return f.GetName().String()
}

func (f *MPIFramework) SetDefault(job *aresv1.AresJob) {
	// jobPhasePolicy
	if job.Spec.JobPhasePolicy == nil {
		job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
			Failed:    [][]commonv1.ReplicaType{{aresv1.RoleLauncher}},
			Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleLauncher}},
		}
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
			job.Spec.JobPhasePolicy.Failed = append(job.Spec.JobPhasePolicy.Failed, []commonv1.ReplicaType{aresv1.RoleWorker})
		}
	}
	// rolePhasePolicy
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
			spec.RolePhasePolicy.NodeDownAsFailed = true
		}
	}
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleLauncher]; ok {
		if spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
		if spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
			spec.RolePhasePolicy.NodeDownAsFailed = true
		}
	}
	// dependence
	if spec, ok := job.Spec.RoleSpecs[aresv1.RoleLauncher]; ok {
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok && spec.Dependence == nil {
			file := aresv1.DefaultMPIHostFilePath
			spec.Dependence = &aresv1.DependenceSpec{
				RoleNames:   []commonv1.ReplicaType{aresv1.RoleWorker},
				RolePhase:   aresv1.RolePhaseRunning,
				MPIHostFile: &file,
			}
		}
	}
	// mpi
	if job.Spec.Framework.MPI == nil {
		job.Spec.Framework.MPI = &aresv1.MPISpec{
			ReplicasPerNode: 1,
			Implementation:  aresv1.OMPI,
		}
	}
}

func (f *MPIFramework) Validate(job *aresv1.AresJob) error {
	return job.Spec.Framework.MPI.Validate()
}

// ReconcileServices: mpi框架无需对launcher和worker创建service
func (f *MPIFramework) ReconcileServices(
	job metav1.Object,
	services []*corev1.Service,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {

	log := util.LoggerForJob(job)
	if rtype == aresv1.RoleLauncher || rtype == aresv1.RoleWorker {
		log.Debugf("I don't need service: %s/%s", f.GetName(), rtype)
		return nil
	}

	return f.Reconciler.ReconcileServices(job, services, rtype, spec)
}
