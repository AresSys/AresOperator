package reconciler

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

/************
 * Phase
 ************/
// IsFinished: 任务是否已完成
func IsFinished(phase aresv1.JobPhase) bool {
	return phase == aresv1.JobPhaseFailed || phase == aresv1.JobPhaseSucceeded
}

// IsRoleFinished: 角色是否已完成
func IsRoleFinished(phase aresv1.RolePhase) bool {
	return phase == aresv1.RolePhaseFailed || phase == aresv1.RolePhaseSucceeded
}

// IsRoleStarted: 角色是否已启动
func IsRoleStarted(phase aresv1.RolePhase) bool {
	return phase == aresv1.RolePhaseRunning || IsRoleFinished(phase)
}

// GetCondition: 获取指定的JobCondition
func GetCondition(status aresv1.AresJobStatus, conditionType commonv1.JobConditionType) *commonv1.JobCondition {
	if status.JobStatus == nil {
		return nil
	}
	for _, cond := range status.Conditions {
		if cond.Type == conditionType {
			return &cond
		}
	}
	return nil
}

/************
 * Labels
 ************/
// GetReplicaType: 获取角色名称
func GetReplicaType(labels map[string]string) string {
	return GetLabelByName(labels, aresv1.ReplicaTypeLabel, commonv1.ReplicaTypeLabel)
}

// GetReplicaIndex: 获取角色索引
func GetReplicaIndex(labels map[string]string) string {
	return GetLabelByName(labels, aresv1.ReplicaIndexLabel, commonv1.ReplicaIndexLabel)
}

// GetLabelByName: 根据label名称获取值
func GetLabelByName(labels map[string]string, candidateNames ...string) string {
	if labels == nil {
		return ""
	}
	for _, name := range candidateNames {
		if value, ok := labels[name]; ok {
			return value
		}
	}
	return ""
}

/************
 * Filter
 ************/
// FilterPods: 按要求过滤Pod
func FilterPods(pods []*corev1.Pod, requires func(p *corev1.Pod) bool) []*corev1.Pod {
	filtered := []*corev1.Pod{}
	for _, pod := range pods {
		if !requires(pod) {
			continue
		}
		filtered = append(filtered, pod)
	}
	return filtered
}

// FilterAlivePods: 过滤存活的Pod
func FilterAlivePods(pods []*corev1.Pod) []*corev1.Pod {
	return FilterPods(pods, func(p *corev1.Pod) bool {
		return p.GetDeletionTimestamp() == nil
	})
}

// FilterTerminatingPods: 过滤非存活的Pod
func FilterTerminatingPods(pods []*corev1.Pod) []*corev1.Pod {
	return FilterPods(pods, func(p *corev1.Pod) bool {
		return p.GetDeletionTimestamp() != nil
	})
}

/************
 * DomainName
 ************/
func GetGeneralSharedName(jobName, rtype string) string {
	n := jobName + "-" + rtype
	return strings.Replace(n, "/", "-", -1)
}

func GetServiceDomainName(jobName, nameSpace string, role commonv1.ReplicaType) string {
	svcName := GetGeneralSharedName(jobName, string(role))
	return fmt.Sprintf("%v.%v.svc.cluster.local", svcName, nameSpace)
}

func GetPodDomainName(jobName, nameSpace string, role commonv1.ReplicaType, index int) string {
	podName := common.GenGeneralName(jobName, string(role), strconv.FormatInt(int64(index), 10))
	serviceDomainName := GetServiceDomainName(jobName, nameSpace, role)
	return fmt.Sprintf("%v.%v", podName, serviceDomainName)
}

func GetRolesDomainNames(job *aresv1.AresJob, roles ...commonv1.ReplicaType) []string {
	res := []string{}
	for _, role := range roles {
		serviceDomainName := GetServiceDomainName(job.Name, job.Namespace, role)
		replicas := job.Spec.GetRoleReplicas(role)
		for i := 0; i < int(replicas); i++ {
			podName := common.GenGeneralName(job.Name, string(role), strconv.FormatInt(int64(i), 10))
			res = append(res, fmt.Sprintf("%s.%s", podName, serviceDomainName))
		}
	}
	return res
}

/************
 * InvalidDefinition
 ************/
// SetInvalidDefinition: 设置任务失败原因为ErrInvalidDefinition
func SetInvalidDefinition(status *aresv1.AresJobStatus, msg string) {
	if status.JobStatus == nil {
		status.JobStatus = &commonv1.JobStatus{}
	}
	util.UpdateJobConditions(status.JobStatus, commonv1.JobFailed, aresv1.ErrInvalidDefinition, msg)
	status.Phase = aresv1.JobPhaseFailed
}

// IsInvalidDefinition: 任务失败原因是否为ErrInvalidDefinition？
func IsInvalidDefinition(status aresv1.AresJobStatus) bool {
	cond := GetCondition(status, commonv1.JobFailed)
	if cond == nil {
		return false
	}
	return cond.Reason == aresv1.ErrInvalidDefinition
}

/************
 * FailedInfo
 ************/
func GetPodFailedInfo(pod *corev1.Pod) string {
	// 1. 获取Pod所在机器
	nodeName := pod.Spec.NodeName
	// 2. 获取终止详情，包括原因等信息
	var terminatedInfo string
	// 2.1 从container状态中汇聚：pod自己异常退出或者oom等情形
	for _, status := range pod.Status.ContainerStatuses {
		terminated := status.State.Terminated
		if terminated == nil {
			continue
		}
		// 示例：{"exitCode":143,"reason":"Error","startedAt":"2021-01-15T07:21:00Z","finishedAt":"2021-01-15T07:25:26Z"}
		info := fmt.Sprintf("%+v", terminated)
		if content, err := json.Marshal(terminated); err == nil {
			info = string(content)
		}
		terminatedInfo = info
		break
	}
	// 2.2 从pod的status中获取：pod被驱逐等情形
	if len(terminatedInfo) == 0 && len(pod.Status.Reason) > 0 {
		// 示例：(Evicted)Usage of EmptyDir volume "group1-storage-1" exceeds the limit "1Mi"
		terminatedInfo = fmt.Sprintf("(%s)%s", pod.Status.Reason, pod.Status.Message)
	}
	// 3. detailedInfo
	infos := []string{
		fmt.Sprintf("nodeName=%s", nodeName),
		fmt.Sprintf("info=%s", terminatedInfo),
	}
	return strings.Join(infos, "; ")
}

/************
 * Sort
 ************/
// 按照角色名和ReplicaIndex，对pods进行排序
func SortPodsByRoleAndIndex(pods []*corev1.Pod) {
	sort.Slice(pods, func(i int, j int) bool {
		// 如果角色不同，按照角色名字典序升序排序
		tpi := GetReplicaType(pods[i].Labels)
		tpj := GetReplicaType(pods[j].Labels)
		if tpi != tpj {
			return tpi < tpj
		}

		// 对相同角色，按照ReplicaIndex整型值从小到大排序
		idxi := GetReplicaIndex(pods[i].Labels)
		idxj := GetReplicaIndex(pods[j].Labels)

		if len(idxi) == 0 || len(idxj) == 0 {
			return i < j
		}
		if len(idxi) == len(idxj) {
			return idxi < idxj
		}
		return len(idxi) < len(idxj)
	})
}
