package reconciler

import (
	"context"
	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

// PodGroupManager
type PodGroupManager struct {
	cli client.Client
}

// NewPodGroupManager: 新建PodGroupManager
func NewPodGroupManager(cli client.Client) *PodGroupManager {
	return &PodGroupManager{
		cli: cli,
	}
}

type ReconcileAction int

const (
	ActionContinue ReconcileAction = iota // 继续
	ActionSkip                            // 跳过
	ActionBreak                           // 终止
)

// SyncFromPodGroup: 从PodGroup中同步信息
func (m *PodGroupManager) SyncFromPodGroup(job *aresv1.AresJob, r *Reconciler) (ReconcileAction, error) {
	if !job.Spec.EnableGangScheduling() {
		return ActionContinue, nil
	}
	podgroup, err := m.GetPodGroupForJob(job)
	if err != nil || podgroup == nil {
		return ActionContinue, nil
	}
	// scheduled?
	if updated, err := m.updateScheduledByPodGroup(job, podgroup); err != nil {
		return 0, err
	} else if updated {
		return ActionBreak, nil
	}
	// evicted?
	if job.Status.Evicted() {
		return ActionSkip, nil
	}
	evicted := m.checkEviction(job, podgroup)
	if evicted == nil {
		return ActionContinue, nil
	}
	if !*evicted {
		return ActionSkip, nil
	}
	if job.Status.Phase != aresv1.JobPhaseFailed {
		r.SetJobPhase(job, &job.Status, aresv1.JobPhaseFailed)
	}
	if err := r.UpdateJobStatusInApiServer(job, job.Status.JobStatus); err != nil {
		return 0, err
	}
	return ActionBreak, nil
}

// PodGroupExists: 判定job管理的PodGroup是否存在
func (m *PodGroupManager) PodGroupExists(job *aresv1.AresJob) (exists bool) {
	_, err := m.GetPodGroupForJob(job)
	if err == nil {
		return true
	}
	if apierrors.IsNotFound(err) {
		return false
	}
	util.LoggerForJob(job).Errorf("failed to check if podgroup exists: %v", err)
	return false
}

// updateScheduledByPodGroup: 根据PodGroup更新Job是否已被调度
func (m *PodGroupManager) updateScheduledByPodGroup(job *aresv1.AresJob, podgroup *scheduling.PodGroup) (updated bool, err error) {
	if cond := GetCondition(job.Status, aresv1.JobScheduled); cond != nil && cond.Status == v1.ConditionTrue {
		return false, nil
	}
	log := util.LoggerForJob(job)
	scheduled, message := IsPodGroupScheduled(podgroup)
	if !scheduled {
		return false, nil
	}
	if err := m.updateJobCondition(job, aresv1.JobScheduled, string(aresv1.JobScheduled), message); err != nil {
		return false, err
	}
	log.Infof("AresJob(%v) %s has been scheduled", job.Spec.FrameworkType, job.Name)
	return true, nil
}

// checkEviction: 检查Job是否被驱逐
func (m *PodGroupManager) checkEviction(job *aresv1.AresJob, podgroup *scheduling.PodGroup) *bool {
	possible, sure := false, true
	// 如果PodGroup被驱逐，则整个Job被驱逐
	evicted, msg := IsPodGroupEvicted(podgroup)
	if evicted {
		m.setEvictedCondition(job, msg)
		return &sure // definitely evicted
	}
	preempted, msg := m.isPodPreempted(job)
	if preempted {
		return &possible // possible evicted
	}
	return nil // not evicted
}

func (m *PodGroupManager) setEvictedCondition(job *aresv1.AresJob, message string) {
	_ = util.UpdateJobConditions(job.Status.JobStatus, commonv1.JobFailed, aresv1.Preempted, message)
}

func (m *PodGroupManager) updateJobCondition(job *aresv1.AresJob, ctype commonv1.JobConditionType, reason, message string) error {
	_ = util.UpdateJobConditions(job.Status.JobStatus, ctype, reason, message)
	if err := m.cli.Status().Update(context.Background(), job); err != nil {
		util.LoggerForJob(job).Errorf("failed to update condition of job: condition=%v, reason=%s, message=%s, error=%v",
			ctype, reason, message, err)
		return err
	}
	return nil
}

// isPodPreempted: 判定Job管理的Pod是否被抢占
func (m *PodGroupManager) isPodPreempted(job *aresv1.AresJob) (bool, string) {
	log := util.LoggerForJob(job)
	// 如果Job的Pods被抢占，则整个Job被驱逐
	// 添加本判断条件是因为PodGroup的Evicted事件可能会晚于Pod的Preempted事件
	pods, err := GetPodsForJob(m.cli, job)
	if err != nil {
		log.Errorf("failed to get pods for job: %v", err)
		return false, ""
	}
	for _, pod := range pods {
		if preempted, msg := IsPodPreempted(pod); preempted {
			return true, msg
		}
	}
	return false, ""
}

// GetPodGroupForJob: 根据job获取其管理的PodGroup
func (m *PodGroupManager) GetPodGroupForJob(job *aresv1.AresJob) (*scheduling.PodGroup, error) {
	pg := &scheduling.PodGroup{}
	err := m.cli.Get(context.Background(), client.ObjectKey{
		Namespace: job.Namespace,
		Name:      job.Name,
	}, pg)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return pg, nil
}

// IsPodPreempted: 根据Pod的Condition标识确定其是否被抢占
func IsPodPreempted(pod *v1.Pod) (bool, string) {
	for _, c := range pod.Status.Conditions {
		if c.Type == v1.PodReady &&
			c.Status == v1.ConditionFalse &&
			c.Reason == aresv1.Preempted {
			return true, c.Message
		}
	}
	return false, ""
}

// IsPodGroupEvicted: 根据PodGroup的Condition标识确定其是否被驱逐
func IsPodGroupEvicted(pg *scheduling.PodGroup) (bool, string) {
	for _, c := range pg.Status.Conditions {
		if c.Type == scheduling.PodGroupConditionType(aresv1.Evicted) &&
			c.Status == v1.ConditionTrue {
			return true, c.Message
		}
	}
	return false, ""
}

// IsPodGroupScheduled: 根据PodGroup的Condition标识确定其是否被调度
func IsPodGroupScheduled(pg *scheduling.PodGroup) (bool, string) {
	for _, c := range pg.Status.Conditions {
		if c.Type == scheduling.PodGroupScheduled &&
			c.Status == v1.ConditionTrue {
			return true, c.Reason
		}
	}
	return false, ""
}

// SyncToPodGroup: 同步信息到PodGroup
func (m *PodGroupManager) SyncToPodGroup(job *aresv1.AresJob, r *Reconciler) error {
	if !job.Spec.EnableGangScheduling() {
		return nil
	}
	podgroup, err := m.GetPodGroupForJob(job)
	if err != nil || podgroup == nil {
		return err
	}
	log := util.LoggerForJob(job)
	if err := patchMetadata(m.cli, job, podgroup); err != nil {
		log.Infof("failed to patch metadata to podgroup <%s/%s>: %v", podgroup.Namespace, podgroup.Name, err)
		return err
	}
	return nil
}
