package controllers

import (
	"context"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	aresv1 "ares-operator/api/v1"
	"ares-operator/cache"
	"ares-operator/reconciler"
)

// OnJobFinished: 任务完成后的处理
func (r *AresJobReconciler) OnJobFinished(job *aresv1.AresJob) error {
	phase := job.Status.Phase
	if !reconciler.IsFinished(phase) {
		return nil
	}
	var (
		duration time.Duration // 要延迟清理的时长
		now      = time.Now()  // 当前时间
		lastTime = now         // 状态变更的时间点
	)

	if phase == aresv1.JobPhaseFailed {
		// 失败后延长该时间后清理
		if job.Spec.TTLSecondsAfterFailed == nil {
			return nil
		}
		duration = time.Duration(*job.Spec.TTLSecondsAfterFailed) * time.Second
		cond := reconciler.GetCondition(job.Status, commonv1.JobFailed)
		if cond != nil && !cond.LastTransitionTime.IsZero() {
			lastTime = cond.LastTransitionTime.Time
		}
	} else {
		// 成功后延长该时间后清理
		ttl := r.Reconciler.Config.TTLSecondsAfterSucceeded()
		if ttl == nil {
			return nil
		}
		duration = *ttl
		cond := reconciler.GetCondition(job.Status, commonv1.JobSucceeded)
		if cond != nil && !cond.LastTransitionTime.IsZero() {
			lastTime = cond.LastTransitionTime.Time
		}
	}
	duration -= now.Sub(lastTime)
	if duration < 0 {
		duration = 0
	}
	util.LoggerForJob(job).Infof("delaying cleanup job because of %v: %v", phase, duration)
	r.CleanUpQueue.AddAfter(types.NamespacedName{
		Namespace: job.Namespace,
		Name:      job.Name,
	}, duration)
	return nil
}

// cleanupNextItem: 清理一个job相关资源
func (r *AresJobReconciler) cleanupNextItem() bool {
	obj, shutdown := r.CleanUpQueue.Get()
	if shutdown {
		// Stop working
		return false
	}
	defer r.CleanUpQueue.Done(obj)

	namespacedName := obj.(types.NamespacedName)
	job, err := r.Reconciler.GetJobFromInformerCache(namespacedName.Namespace, namespacedName.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			r.CleanUpQueue.Forget(obj)
		} else {
			r.CleanUpQueue.AddRateLimited(obj)
		}
		return true
	}
	aresjob := job.(*aresv1.AresJob)
	framework := r.getFramework(aresjob.Spec.FrameworkType)
	if framework == nil {
		r.CleanUpQueue.Forget(obj)
		return true
	}
	if err := framework.CleanUpResources(aresjob); err != nil {
		r.CleanUpQueue.AddRateLimited(obj)
	} else {
		r.CleanUpQueue.Forget(obj)
	}
	return true
}

// HandleRolePhases: 处理角色状态
func (r *AresJobReconciler) HandleRolePhases(old, job *aresv1.AresJob) error {
	if !job.Spec.HasDependence() {
		return nil
	}
	framework := r.getFramework(job.Spec.FrameworkType)
	if framework == nil {
		return nil
	}
	for roleName, roleStatus := range job.Status.RoleStatuses {
		oldPhase := aresv1.RolePhasePending
		if status, ok := old.Status.RoleStatuses[roleName]; ok {
			oldPhase = status.Phase
		}
		// 理论oldPhase != roleStatus.Phase时需要调用OnRolePhaseChanged；
		// 为保险起见，当前状态为Running时也调用OnRolePhaseChanged
		if oldPhase == roleStatus.Phase && roleStatus.Phase != aresv1.RolePhaseRunning {
			continue
		}
		if err := framework.OnRolePhaseChanged(job, roleName, roleStatus.Phase); err != nil {
			return err
		}
	}
	return nil
}

// finalize: 删除前清理
func (r *AresJobReconciler) finalize(job *aresv1.AresJob) error {
	r.Reconciler.CleanForJob(job)
	// 清理外部数据
	if !controllerutil.ContainsFinalizer(job, aresv1.AresJobFinalizer) {
		return nil
	}
	util.LoggerForJob(job).Info("trying to finalize...")
	if err := r.deleteJobCache(types.NamespacedName{
		Namespace: job.Namespace,
		Name:      job.Name,
	}); err != nil {
		return err
	}
	controllerutil.RemoveFinalizer(job, aresv1.AresJobFinalizer)
	return r.Reconciler.Update(context.Background(), job)
}

// OnJobDeleted: 任务删除后的处理
func (r *AresJobReconciler) deleteJobCache(nn types.NamespacedName) error {
	c := r.Reconciler.Cache
	if c == nil {
		return nil
	}
	// delete job cache
	key := cache.NamespacedNameToKey(nn.Namespace, nn.Name)
	log := util.LoggerForKey(key)
	if err := c.DeleteJobCache(key); err != nil {
		log.Errorf("failed to delete job cache: %v", err)
		return err
	}
	return nil
}
