package common

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler"
)

type Framework interface {
	aresv1.Controller

	// Framework属性
	GetName() aresv1.FrameworkType
	SetController(c aresv1.Controller)

	// AresJob属性
	SetDefault(job *aresv1.AresJob)
	Validate(job *aresv1.AresJob) error

	// 框架流程处理
	// 1. 事件处理
	AddObject(job *aresv1.AresJob, obj metav1.Object)
	DeleteObject(job *aresv1.AresJob, obj metav1.Object)
	OnRolePhaseChanged(job *aresv1.AresJob, rtype commonv1.ReplicaType, phase aresv1.RolePhase) error
	// 2. 资源清理
	CleanUpResources(job *aresv1.AresJob) error
}

type FrameworkBuilder func(r *reconciler.Reconciler, mgr ctrl.Manager) Framework
