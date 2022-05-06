package reconciler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/util"
	"github.com/kubeflow/common/pkg/util/train"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aresv1 "ares-operator/api/v1"
	aresmeta "ares-operator/metadata"
)

const (
	// restartPolicyOverwrittenReason is the warning reason when the restart
	// policy is overwritten in pod template.
	restartPolicyOverwrittenReason = "RestartPolicyOverwritten"
	// podFailedReason is the normal reason when the pod is failed.
	podFailedReason  = "PodFailed"
	podMissingReason = "PodMissing"
	nodeDownReason   = "PossibleNodeDown"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *Reconciler) GetPodsForJob(job interface{}) ([]*corev1.Pod, error) {
	obj, err := meta.Accessor(job)
	if err != nil {
		return nil, err
	}
	return GetPodsForJob(r.Client, obj.(*aresv1.AresJob))
}

func GetPodsForJob(c client.Client, job *aresv1.AresJob) ([]*corev1.Pod, error) {
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	if err := c.List(context.Background(), podlist, client.InNamespace(job.GetNamespace()),
		client.MatchingFields{aresv1.JobOwnerKey: job.GetName()}); err != nil {
		return nil, err
	}

	return convertPodList(podlist.Items), nil
}

// GetPodsForReplica: 根据指定job的角色获取pod列表
func (r *Reconciler) GetPodsForReplica(job *aresv1.AresJob, rtype commonv1.ReplicaType) ([]*corev1.Pod, error) {
	podlist := &corev1.PodList{}
	if err := r.List(context.Background(), podlist, client.InNamespace(job.Namespace),
		client.MatchingFields{aresv1.ReplicaOwnerKey: aresv1.MetaReplicaKeyFormat(job.Name, rtype)}); err != nil {
		return nil, err
	}
	return convertPodList(podlist.Items), nil
}

// GetPodsForNode: 获取指定node上的pod列表
func (r *Reconciler) GetPodsForNode(name string) ([]*corev1.Pod, error) {
	podlist := &corev1.PodList{}
	if err := r.List(context.Background(), podlist,
		client.MatchingFields{aresv1.NodeNameKey: name}); err != nil {
		return nil, err
	}
	return convertPodList(podlist.Items), nil
}

// convertPodList convert pod list to pod point list
func convertPodList(list []corev1.Pod) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// FilterPodsForReplicaType returns pods belong to a replicaType.
func (r *Reconciler) FilterPodsForReplicaType(pods []*corev1.Pod, replicaType string) ([]*corev1.Pod, error) {
	replicaSelector := labels.SelectorFromSet(labels.Set{aresv1.ReplicaTypeLabel: replicaType})
	deprecatedSelector := labels.SelectorFromSet(labels.Set{commonv1.ReplicaTypeLabel: replicaType})

	var result []*corev1.Pod
	for _, pod := range pods {
		if !deprecatedSelector.Matches(labels.Set(pod.Labels)) && !replicaSelector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		result = append(result, pod)
	}
	return result, nil
}

// getPodSlices returns a slice, which element is the slice of pod.
// It gives enough information to caller to make decision to up/down scale resources.
func (r *Reconciler) GetPodSlices(pods []*corev1.Pod, replicas int, logger *log.Entry) [][]*corev1.Pod {
	podSlices := make([][]*corev1.Pod, calculatePodSliceSize(pods, replicas))
	for _, pod := range pods {
		replicaIndex := GetReplicaIndex(pod.Labels)
		if len(replicaIndex) == 0 {
			logger.Warning("The pod do not have the index label.")
			continue
		}
		index, err := strconv.Atoi(replicaIndex)
		if err != nil {
			logger.Warningf("Error when strconv.Atoi: %v", err)
			continue
		}
		if index < 0 || index >= replicas {
			logger.Warningf("The label index is not expected: %d, pod: %s/%s", index, pod.Namespace, pod.Name)
		}

		podSlices[index] = append(podSlices[index], pod)
	}
	return podSlices
}

// calculatePodSliceSize compare max pod index with desired replicas and return larger size
func calculatePodSliceSize(pods []*corev1.Pod, replicas int) int {
	size := 0
	for _, pod := range pods {
		replicaIndex := GetReplicaIndex(pod.Labels)
		if len(replicaIndex) == 0 {
			continue
		}
		index, err := strconv.Atoi(replicaIndex)
		if err != nil {
			continue
		}
		size = common.MaxInt(size, index)
	}

	// size comes from index, need to +1 to indicate real size
	return common.MaxInt(size+1, replicas)
}

// ReconcilePods checks and updates pods for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods.
func (r *Reconciler) ReconcilePods(
	job interface{},
	jobStatus *commonv1.JobStatus,
	pods []*corev1.Pod,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	aresjob, ok := job.(*aresv1.AresJob)
	if !ok {
		return fmt.Errorf("job is not an AresJob")
	}
	jobKey, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}
	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, string(rtype))

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))
	logger := util.LoggerForReplica(aresjob, rt)
	// Get all pods for the type rt.
	pods, err = r.FilterPodsForReplicaType(pods, rt)
	if err != nil {
		return err
	}
	numReplicas := int(*spec.Replicas)
	var masterRole bool

	reconcilingPods := FilterAlivePods(pods)
	validatedPods := FilterTerminatingPods(pods)
	if spec.RestartPolicy == commonv1.RestartPolicyNever {
		reconcilingPods = pods
		validatedPods = []*corev1.Pod{}
	}
	expectedCreation, expectedDeletion := 0, 0

	// GetPodSlices will return enough information here to make decision to add/remove/update resources.
	//
	// For example, let's assume we have pods with replica-index 0, 1, 2
	// If replica is 4, return a slice with size 4. [[0],[1],[2],[]], a pod with replica-index 3 will be created.
	//
	// If replica is 1, return a slice with size 3. [[0],[1],[2]], pod with replica-index 1 and 2 are out of range and will be deleted.
	podSlices := r.GetPodSlices(reconcilingPods, numReplicas, logger)
	for index, podSlice := range podSlices {
		if len(podSlice) > 1 {
			logger.Warningf("We have too many pods for %s %d", rt, index)
		} else if len(podSlice) == 0 {
			rolePhase := aresjob.Status.RolePhase(rtype)
			if spec.RestartPolicy == commonv1.RestartPolicyNever && IsRoleStarted(rolePhase) {
				logger.Warningf("pod of index %d is missing: phase=%v", index, rolePhase)
			} else {
				logger.Infof("Need to create new pod: %s-%d", rt, index)

				// check if this replica is the master role
				masterRole = r.Controller.IsMasterRole(replicas, rtype, index)
				err = r.CreateNewPod(job, rt, strconv.Itoa(index), spec, masterRole, replicas)
				if err != nil {
					return err
				}
				expectedCreation += 1
			}
		} else {
			// Check the status of the current pod.
			pod := podSlice[0]

			// check if the index is in the valid range, if not, we should kill the pod
			if index < 0 || index >= numReplicas {
				err = r.PodControl.DeletePod(pod.Namespace, pod.Name, aresjob)
				if err != nil {
					return err
				}
				r.Expectations.RaiseExpectations(expectationPodsKey, 0, 1)
				expectedDeletion += 1
			}

			// NOTE: 已改动
			if pod.Status.Phase == corev1.PodFailed {
				// Get the exit code of the container.
				var exitCode int32 = 0xbeef // magic number
				for _, status := range pod.Status.ContainerStatuses {
					state := status.State
					if state.Terminated != nil {
						exitCode = state.Terminated.ExitCode
						break
					}
				}
				info := GetPodFailedInfo(pod)
				logger.Infof("pod %s.%s is failed: %s", pod.Namespace, pod.Name, info)
				r.Recorder.Eventf(aresjob, corev1.EventTypeNormal, podFailedReason, "Pod %s is failed: %s", pod.Name, info)

				// Check if the pod is retryable.
				if spec.RestartPolicy == commonv1.RestartPolicyAlways || // Deployment控制器的行为
					spec.RestartPolicy == commonv1.RestartPolicyOnFailure || // Job的行为
					spec.RestartPolicy == commonv1.RestartPolicyExitCode && train.IsRetryableExitCode(exitCode) {
					logger.Infof("need to restart the pod: %v.%v", pod.Namespace, pod.Name)
					if err := r.PodControl.DeletePod(pod.Namespace, pod.Name, aresjob); err != nil {
						return err
					}
					r.Expectations.RaiseExpectations(expectationPodsKey, 0, 1)
					expectedDeletion += 1
				} else { // not retryable
					validatedPods = append(validatedPods, pod)
				}
			} else { // not failed
				validatedPods = append(validatedPods, pod)
			}

		}
	}

	aresjob.Status.JobStatus = jobStatus
	r.UpdateRoleReplicas(aresjob, rtype, validatedPods)
	if expectedCreation == 0 && expectedDeletion == 0 {
		r.UpdateRolePhase(aresjob, rtype, validatedPods)
	} else {
		logger.Infof("skip update role phase for %s, expected pod: creation=%d, deletion=%d", rt, expectedCreation, expectedDeletion)
	}
	return nil
}

// GeneratePodName generates pod name for given parameters
func (r *Reconciler) GeneratePodName(jobName, roleName, index string) string {
	return common.GenGeneralName(jobName, roleName, index)
}

// CreateNewPod: creates a new pod for the given index and type.
func (r *Reconciler) CreateNewPod(j interface{}, rt, index string, spec *commonv1.ReplicaSpec, masterRole bool,
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec) error {

	job := j.(*aresv1.AresJob)
	jobKey, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Set type and index for the worker.
	labels := r.GenLabels(job.GetName())
	labels[aresv1.ReplicaTypeLabel] = rt
	labels[aresv1.ReplicaIndexLabel] = index

	if masterRole {
		labels[aresv1.JobRoleLabel] = string(aresv1.RoleMaster)
	}

	podTemplate := spec.Template.DeepCopy()

	// Set name for the template.
	name := r.FrameworkController.GeneratePodName(job.GetName(), rt, index)
	if spec.RestartPolicy == commonv1.RestartPolicyNever {
		podTemplate.Name = name
	} else {
		podTemplate.GenerateName = name + "-"
	}

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	for key, value := range labels {
		podTemplate.Labels[key] = value
	}
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}
	for key, value := range job.Annotations {
		if aresmeta.IsJobLevelKey(key) {
			podTemplate.Annotations[key] = value
			podTemplate.Labels[key] = value
		}
	}

	podTemplate.Spec.Hostname = name
	podTemplate.Spec.Subdomain = GetGeneralSharedName(job.GetName(), rt)

	if err := r.Controller.SetClusterSpec(job, podTemplate, rt, index); err != nil {
		return err
	}

	r.setRestartPolicy(job, rt, podTemplate, spec)

	// NOTE: 已改动
	r.SetScheduler(job, podTemplate)
	if err := r.SetDependence(job, podTemplate, rt, index); err != nil {
		return err
	}

	expectationPodsKey := expectation.GenExpectationPodsKey(jobKey, rt)
	r.Expectations.RaiseExpectations(expectationPodsKey, 1, 0)
	controllerRef := r.GenOwnerReference(job)
	err = r.PodControl.CreatePodsWithControllerRef(job.GetNamespace(), podTemplate, job, controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Pod is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the pod keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// pod when the expectation expires.
		return nil
	} else if err != nil {
		r.Expectations.CreationObserved(expectationPodsKey)
		return err
	}
	return nil
}

func (r *Reconciler) setRestartPolicy(job *aresv1.AresJob, rt string, podTemplateSpec *corev1.PodTemplateSpec, spec *commonv1.ReplicaSpec) {
	// This is necessary since restartPolicyExitCode is not supported in v1.PodTemplateSpec
	var restartPolicy corev1.RestartPolicy
	if spec.RestartPolicy == commonv1.RestartPolicyExitCode {
		restartPolicy = corev1.RestartPolicyNever
	} else {
		restartPolicy = corev1.RestartPolicy(spec.RestartPolicy)
	}
	// Submit a warning event if the user specifies restart policy for
	// the pod template. We recommend to set it from the replica level.
	original := podTemplateSpec.Spec.RestartPolicy
	if len(original) > 0 && original != restartPolicy {
		msg := fmt.Sprintf("Restart policy for %s is overwritten: %s -> %s", rt, original, restartPolicy)
		util.LoggerForReplica(job, rt).Warning(msg)
		r.Recorder.Event(job, corev1.EventTypeWarning, restartPolicyOverwrittenReason, msg)
	}
	podTemplateSpec.Spec.RestartPolicy = restartPolicy
}

// SetScheduler: 设置调度相关
func (r *Reconciler) SetScheduler(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec) {
	// if gang-scheduling is enabled, we set the SchedulerName to "volcano".
	if job.Spec.EnableGangScheduling() {
		podTemplate.Spec.SchedulerName = aresv1.GangSchedulerName
		if podTemplate.Annotations == nil {
			podTemplate.Annotations = map[string]string{}
		}
		podTemplate.Annotations[aresv1.GangSchedulingPodGroupAnnotation] = job.GetName()
	}
}

// SetDependence: 设置依赖相关
func (r *Reconciler) SetDependence(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	log := util.LoggerForReplica(job, rtype)
	spec := job.Spec.RoleSpecs[commonv1.ReplicaType(rtype)]
	if spec == nil || spec.Dependence == nil {
		return nil
	}
	var err error
	if spec.Dependence.MPIHostFile == nil {
		err = r.AddWaiter(job, podTemplate, rtype, index)
	} else {
		err = r.AddMPIInitializer(job, podTemplate, rtype, index)
	}
	if err != nil {
		log.Errorf("failed to add waiter: %v", err)
	}
	return err
}

// GetPort: 获取端口
func (r *Reconciler) GetPort(job *aresv1.AresJob, rtype commonv1.ReplicaType) (int32, error) {
	containers := job.Spec.RoleSpecs[rtype].Template.Spec.Containers
	for _, container := range containers {
		if container.Name != r.Controller.GetDefaultContainerName() {
			continue
		}
		for _, port := range container.Ports {
			if port.Name == r.Controller.GetDefaultContainerPortName() {
				return port.ContainerPort, nil
			}
		}
	}
	return 0, aresv1.ErrPortNotFound
}

// SyncToPod: 同步信息到Pod
func (r *Reconciler) SyncToPod(job *aresv1.AresJob) error {
	log := util.LoggerForJob(job)
	pods, err := GetPodsForJob(r.Client, job)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		if err := patchMetadata(r.Client, job, pod); err != nil {
			log.Infof("failed to patch metadata to pod <%s/%s>: %v", pod.Namespace, pod.Name, err)
			return err
		}
	}
	return nil
}
