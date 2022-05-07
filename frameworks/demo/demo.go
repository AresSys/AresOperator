package demo

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
)

var _ fcommon.Framework = &DemoFramework{}

var DemoBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &DemoFramework{Reconciler: r}
}

type DemoFramework struct {
	*reconciler.Reconciler
}

func (f *DemoFramework) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkDemo
}

func (f *DemoFramework) SetDefault(job *aresv1.AresJob) {
}

func (f *DemoFramework) Validate(job *aresv1.AresJob) error {
	return nil
}

func (f *DemoFramework) ReconcileJobs(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus commonv1.JobStatus, runPolicy *commonv1.RunPolicy) error {
	f.Log.Info("ReconcileJobs", "framework", f.GetName())
	return f.Controller.UpdateJobStatus(job, replicas, &jobStatus)
}

func (f *DemoFramework) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	f.Log.Info("UpdateJobStatus", "framework", f.GetName())
	return nil
}

func (f *DemoFramework) ReconcileServices(job metav1.Object, services []*corev1.Service, rtype commonv1.ReplicaType, spec *commonv1.ReplicaSpec) error {
	f.Log.Info("ReconcileServices", "framework", f.GetName())
	return nil
}
