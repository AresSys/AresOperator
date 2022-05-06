package reconciler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	aresv1 "ares-operator/api/v1"
	"ares-operator/cache"
	"ares-operator/reconciler/node"
)

// initializeReplicaStatuses initializes the ReplicaStatuses for replica.
func initializeReplicaStatuses(status *aresv1.AresJobStatus, rtype commonv1.ReplicaType) {
	if status.RoleStatuses == nil {
		status.RoleStatuses = map[commonv1.ReplicaType]*aresv1.RoleStatus{}
	}
	if status.RoleStatuses[rtype] == nil {
		status.RoleStatuses[rtype] = &aresv1.RoleStatus{Phase: aresv1.RolePhasePending}
	}
	status.RoleStatuses[rtype].ReplicaStatus = aresv1.ReplicaStatus{
		ReplicaStatus: &commonv1.ReplicaStatus{},
	}
}

// updateRoleReplicaStatuses updates the Role ReplicaStatuses according to the pod.
func updateRoleReplicaStatuses(status *aresv1.AresJobStatus, rtype commonv1.ReplicaType, pod *corev1.Pod) {
	switch pod.Status.Phase {
	case "", corev1.PodPending:
		status.RoleStatuses[rtype].Pending++
	case corev1.PodRunning:
		status.RoleStatuses[rtype].Active++
	case corev1.PodSucceeded:
		status.RoleStatuses[rtype].Succeeded++
	case corev1.PodFailed:
		status.RoleStatuses[rtype].Failed++
	case corev1.PodUnknown:
		status.RoleStatuses[rtype].Unknown++
	}
}

func (r *Reconciler) updateReplicaStatus(job *aresv1.AresJob, roleName commonv1.ReplicaType, pod *corev1.Pod) {
	roleSpec := job.Spec.RoleSpecs[roleName]
	status := &job.Status
	if pod.GetDeletionTimestamp() != nil {
		status.RoleStatuses[roleName].Terminating++
		return
	}
	if len(pod.Spec.NodeName) > 0 && roleSpec.RolePhasePolicy.NodeDownAsFailed {
		if nodeDown, _ := r.NodeManager.IsNodeDown(pod.Spec.NodeName); nodeDown {
			status.RoleStatuses[roleName].Failed++
			return
		}
	}
	updateRoleReplicaStatuses(&job.Status, roleName, pod)
}

/***********
 * RolePhase
 ***********/
// UpdateRoleReplicas: 更新角色副本状态
func (r *Reconciler) UpdateRoleReplicas(job *aresv1.AresJob, roleName commonv1.ReplicaType, pods []*corev1.Pod) {
	jobStatus := &job.Status
	// rolePhase
	initializeReplicaStatuses(jobStatus, roleName)
	for _, pod := range pods {
		r.updateReplicaStatus(job, roleName, pod)
	}
	// 赋值给JobStatus
	if jobStatus.JobStatus.ReplicaStatuses == nil {
		jobStatus.JobStatus.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}
	jobStatus.JobStatus.ReplicaStatuses[roleName] = jobStatus.RoleStatuses[roleName].ReplicaStatus.ReplicaStatus
}

// UpdateRolePhase: 更新角色状态信息
func (r *Reconciler) UpdateRolePhase(job *aresv1.AresJob, roleName commonv1.ReplicaType, pods []*corev1.Pod) {
	roleSpec := job.Spec.RoleSpecs[roleName]
	jobStatus := &job.Status
	log := util.LoggerForReplica(job, string(roleName))

	rolePhase := r.GetRolePhase(roleName, *roleSpec, job.Status)
	roleStatus := jobStatus.RoleStatuses[roleName]
	if rolePhase != roleStatus.Phase {
		msg := fmt.Sprintf("Phase of role %s of AresJob(%v) %s has changed: %s -> %s", roleName, job.Spec.FrameworkType, job.Name, roleStatus.Phase, rolePhase)
		roleStatus.Phase = rolePhase
		roleStatus.LastTransitionTime = metav1.Now()
		if rolePhase == aresv1.RolePhaseFailed {
			roleStatus.Reason, roleStatus.Message = extractFailedReason(r.NodeManager, pods, int(*roleSpec.Replicas))
		}
		r.Recorder.Event(job, corev1.EventTypeNormal, RolePhaseTransitionReason, msg)
		log.Infof("%s; reason=(%s)%s", msg, roleStatus.Reason, roleStatus.Message)
	}
}

func extractFailedReason(m *node.Manager, pods []*corev1.Pod, expected int) (string, string) {
	alivePods := FilterAlivePods(pods)
	// podFailedReason
	for _, pod := range alivePods {
		phase := pod.Status.Phase
		if phase != corev1.PodFailed {
			continue
		}
		return podFailedReason, fmt.Sprintf("pod=%s; %s", pod.Name, GetPodFailedInfo(pod))
	}

	// nodeDownReason
	for _, pod := range alivePods {
		nodeName := pod.Spec.NodeName
		if len(nodeName) == 0 {
			continue
		}
		if nodeDown, reason := m.IsNodeDown(nodeName); nodeDown {
			return nodeDownReason, fmt.Sprintf("pod=%s, node=%s: %s", pod.Name, nodeName, reason)
		}
	}

	// podMissingReason
	// case1: terminating
	terminating := []string{}
	terminatingDetail := []string{}
	indexSet := sets.NewString()
	for _, pod := range pods {
		index := GetReplicaIndex(pod.Labels)
		indexSet.Insert(index)
		if pod.GetDeletionTimestamp() != nil {
			terminating = append(terminating, index)
			if len(terminatingDetail) < 3 {
				terminatingDetail = append(terminatingDetail, fmt.Sprintf("%s(%s)", pod.Name, pod.Spec.NodeName))
			}
		}
	}
	if len(terminating) > 0 {
		sort.Strings(terminating)
		return podMissingReason, fmt.Sprintf("total %d pods were terminating: %s; %s...",
			len(terminating), strings.Join(terminating, ","), strings.Join(terminatingDetail, ","))
	}
	// case2: missing
	missing := []string{}
	for i := 0; i < expected; i++ {
		if indexSet.Has(strconv.Itoa(i)) {
			continue
		}
		missing = append(missing, strconv.Itoa(i))
	}
	if len(missing) > 0 {
		return podMissingReason, fmt.Sprintf("total %d pods were missing: %s", len(missing), strings.Join(missing, ","))
	}
	return "", ""
}

// GetRolePhase: 获取角色状态
func (r *Reconciler) GetRolePhase(rtype commonv1.ReplicaType, spec aresv1.RoleSpec, aresStatus aresv1.AresJobStatus) aresv1.RolePhase {
	policy := spec.RolePhasePolicy
	replicas := *spec.Replicas
	status := aresStatus.RoleStatuses[rtype]
	lastPhase := aresStatus.RoleStatuses[rtype].Phase
	// failed
	if policy.Failed != nil {
		switch *policy.Failed {
		case aresv1.RolePhaseConditionAnyPod:
			if status.Failed > 0 {
				return aresv1.RolePhaseFailed
			}
		case aresv1.RolePhaseConditionAllPods:
			if status.Failed >= replicas {
				return aresv1.RolePhaseFailed
			}
		}
	}

	// succeeded
	if policy.Succeeded != nil {
		switch *policy.Succeeded {
		case aresv1.RolePhaseConditionAnyPod:
			if status.Succeeded > 0 {
				return aresv1.RolePhaseSucceeded
			}
		case aresv1.RolePhaseConditionAllPods:
			if status.Succeeded >= replicas {
				return aresv1.RolePhaseSucceeded
			}
		}
	}

	// running
	if status.Active >= replicas {
		return aresv1.RolePhaseRunning
	}

	// starting
	if lastPhase == aresv1.RolePhasePending {
		return aresv1.RolePhaseStarting
	}

	actual := status.Active + status.Succeeded + status.Failed + status.Unknown + status.Pending
	if spec.RestartPolicy == commonv1.RestartPolicyNever {
		// Running后实例数量减少，说能可能Pod被驱逐或删除
		if lastPhase == aresv1.RolePhaseRunning && actual < replicas {
			return aresv1.RolePhaseFailed
		}
	}
	return lastPhase
}

/***********
 * JobPhase
 ***********/
// UpdatePhases: 更新任务状态信息
func (r *Reconciler) UpdatePhases(job *aresv1.AresJob, roleSpecs map[commonv1.ReplicaType]*aresv1.RoleSpec, jobStatus *aresv1.AresJobStatus) error {
	// jobPhase
	if len(jobStatus.Phase) == 0 {
		jobStatus.Phase = aresv1.JobPhasePending
	}
	if job.Spec.JobPhasePolicy == nil {
		job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{}
	}
	phase := r.GetJobPhase(jobStatus.Phase, *job.Spec.JobPhasePolicy, jobStatus.RoleStatuses)
	if phase != jobStatus.Phase {
		msg := fmt.Sprintf("AresJob %s is %s.", job.Name, phase)
		_ = util.UpdateJobConditions(jobStatus.JobStatus, aresv1.JobPhaseToCondition(phase), phase.String(), msg)
		r.SetJobPhase(job, jobStatus, phase)
	}
	return nil
}

// SetJobPhase: 设置任务状态
func (r *Reconciler) SetJobPhase(job *aresv1.AresJob, jobStatus *aresv1.AresJobStatus, phase aresv1.JobPhase) {
	now := metav1.Now()
	if IsFinished(phase) && jobStatus.CompletionTime == nil {
		jobStatus.CompletionTime = &now
	}
	if phase == aresv1.JobPhaseRunning && jobStatus.RunningTime == nil {
		jobStatus.RunningTime = &now
	}
	jobStatus.Phase = phase
	msg := fmt.Sprintf("Phase of AresJob(%v) %s has changed: %s -> %s", job.Spec.FrameworkType, job.Name, jobStatus.Phase, phase)
	r.Recorder.Event(job, corev1.EventTypeNormal, JobPhaseTransitionReason, msg)
	util.LoggerForJob(job).Info(msg)
}

// GetJobPhase: 获取任务状态
func (r *Reconciler) GetJobPhase(lastPhase aresv1.JobPhase, policy aresv1.JobPhasePolicy, roleStatuses map[commonv1.ReplicaType]*aresv1.RoleStatus) aresv1.JobPhase {
	satisfyPolicy := func(conds [][]commonv1.ReplicaType, rolePhase aresv1.RolePhase) bool {
		for _, roleNames := range conds {
			satisfied := true
			for _, roleName := range roleNames {
				if roleStatuses[roleName].Phase != rolePhase {
					satisfied = false
					break
				}
			}
			if satisfied {
				return true
			}
		}
		return false
	}
	if satisfyPolicy(policy.Failed, aresv1.RolePhaseFailed) {
		return aresv1.JobPhaseFailed
	}
	if satisfyPolicy(policy.Succeeded, aresv1.RolePhaseSucceeded) {
		return aresv1.JobPhaseSucceeded
	}
	var roleNames []commonv1.ReplicaType
	for roleName := range roleStatuses {
		roleNames = append(roleNames, roleName)
	}
	if satisfyPolicy([][]commonv1.ReplicaType{roleNames}, aresv1.RolePhaseRunning) {
		return aresv1.JobPhaseRunning
	}
	if lastPhase == aresv1.JobPhasePending {
		return aresv1.JobPhaseStarting
	}
	return lastPhase
}

/******************
 * RolePhaseChange
 ******************/
// OnRolePhaseChanged: 角色状态变化处理
func (r *Reconciler) OnRolePhaseChanged(job *aresv1.AresJob, rtype commonv1.ReplicaType, phase aresv1.RolePhase) error {
	c := r.Cache
	if c == nil {
		return nil
	}
	log := util.LoggerForReplica(job, string(rtype))
	pods, err := r.GetPodsForReplica(job, rtype)
	if err != nil {
		log.Errorf("failed to get pods for replica: %v", err)
		return err
	}
	key := cache.NamespacedNameToKey(job.Namespace, job.Name)
	data := BuildRoleCache(job, rtype, phase, pods)
	if err := c.SetRoleCache(key, rtype, data); err != nil {
		log.Errorf("failed to handle phase change: %v", err)
		return err
	}
	return nil
}

// BuildRoleCache: 构建RoleCache对象
func BuildRoleCache(job *aresv1.AresJob, rtype commonv1.ReplicaType, phase aresv1.RolePhase, pods []*corev1.Pod) *cache.RoleCache {
	pods = FilterAlivePods(pods)
	// 获取pod的ip列表
	ips := []string{}
	fn := func(pod *corev1.Pod) string {
		return pod.Status.PodIP // 使用PodIP
	}
	if job.Spec.RoleSpecs[rtype].Template.Spec.HostNetwork {
		fn = func(pod *corev1.Pod) string {
			return pod.Status.HostIP // 使用HostIP
		}
	}
	SortPodsByRoleAndIndex(pods)
	for _, pod := range pods {
		if GetReplicaType(pod.Labels) != string(rtype) {
			continue
		}
		ip := fn(pod)
		if len(ip) == 0 {
			continue
		}
		ips = append(ips, ip)
	}
	return cache.NewRoleCache(phase, cache.WithIPs(ips))
}
