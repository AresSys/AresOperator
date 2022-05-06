package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"time"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	aresv1 "ares-operator/api/v1"
)

const (
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"

	JobPhaseTransitionReason  = "JobPhaseTransition"
	RolePhaseTransitionReason = "RolePhaseTransition"
)

func (r *Reconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	log := r.Log.WithValues("aresjob", namespacedName)

	job := &aresv1.AresJob{}

	err := r.Get(context.Background(), namespacedName, job)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "failed to get AresJob from informer cache", "namespace", namespace, "name", name)
		}
		return nil, err
	}

	return job, nil
}

// GetJobFromAPIClient 实际上没有被调用
func (r *Reconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return r.GetJobFromInformerCache(namespace, name)
}

func (r *Reconciler) DeleteJob(job interface{}) error {
	aresJob, ok := job.(*aresv1.AresJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of AresJob", job)
	}

	log := util.LoggerForJob(aresJob)
	if err := r.Delete(context.Background(), aresJob); err != nil {
		r.Recorder.Eventf(aresJob, corev1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		log.Errorf("failed to delete job %s/%s, %v", aresJob.Namespace, aresJob.Name, err)
		return err
	}

	r.Recorder.Eventf(aresJob, corev1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", aresJob.Name)
	log.Infof("job %s/%s has been deleted", aresJob.Namespace, aresJob.Name)

	return nil
}

// UpdateJobStatus: 更新job状态
func (r *Reconciler) UpdateJobStatus(job interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	aresJob, ok := job.(*aresv1.AresJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of AresJob", job)
	}
	log := util.LoggerForJob(aresJob)

	aresJob.Status.JobStatus = jobStatus
	if err := r.UpdatePhases(aresJob, aresJob.Spec.RoleSpecs, &aresJob.Status); err != nil {
		log.Errorf("failed to UpdatePhases: %v", err)
		return err
	}
	log.Debugf("UpdateJobStatus: name=%s, phase=%s", aresJob.Name, aresJob.Status.Phase)
	return nil
}

func (r *Reconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	aresJob, ok := job.(*aresv1.AresJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of AresJob", job)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&aresJob.Status.JobStatus, jobStatus) {
		aresJob = aresJob.DeepCopy()
		aresJob.Status.JobStatus = jobStatus.DeepCopy()
	}

	result := r.Status().Update(context.Background(), aresJob)
	log := util.LoggerForJob(aresJob)
	log.Infof("trying to update job status: %+v", getPhases(aresJob))
	if result != nil {
		log.Error(result, "failed to update AresJob conditions in the API server")
		return result
	}

	return nil
}

func getPhases(job *aresv1.AresJob) map[commonv1.ReplicaType]aresv1.RolePhase {
	phases := map[commonv1.ReplicaType]aresv1.RolePhase{}
	for role, status := range job.Status.RoleStatuses {
		phases[role] = status.Phase
	}
	return phases
}

// CleanUpResources: 清理job相关的资源信息
func (r *Reconciler) CleanUpResources(job *aresv1.AresJob) error {
	log := util.LoggerForJob(job)
	// pods
	pods, err := r.GetPodsForJob(job)
	if err != nil {
		log.Warnf("GetPodsForJob error: %v", err)
		return err
	}
	for _, pod := range pods {
		if err := r.PodControl.DeletePod(pod.Namespace, pod.Name, job); err != nil {
			log.Warningf("failed to delete pod %s: %v", pod.Name, err)
			return err
		}
	}

	// services
	services, err := r.GetServicesForJob(job)
	if err != nil {
		log.Warnf("GetServicesForJob error: %v", err)
		return err
	}
	for _, svc := range services {
		if err := r.ServiceControl.DeleteService(svc.Namespace, svc.Name, job); err != nil {
			log.Warningf("failed to delete service %s: %v", svc.Name, err)
			return err
		}
	}

	// podgroup
	if job.Spec.EnableGangScheduling() {
		if err := r.JobController.DeletePodGroup(job); err != nil {
			log.Warningf("failed to delete podgroup %s: %v", job.Name, err)
			return err
		}
	}
	return nil
}

// SyncFromJob: 同步Job相关信息到其关联的对象，例如PodGroup和Pod
func (r *Reconciler) SyncFromJob(job *aresv1.AresJob) error {
	if err := r.PodGroupManager.SyncToPodGroup(job, r); err != nil {
		return err
	}
	if err := r.SyncToPod(job); err != nil {
		return err
	}
	return nil
}

// ReconcileJobs checks and updates replicas for each given ReplicaSpec.
// It will requeue the job in case of an error while creating/deleting pods/services.
func (r *Reconciler) ReconcileJobs(
	j interface{},
	replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	jobStatus commonv1.JobStatus,
	runPolicy *commonv1.RunPolicy) error {
	job := j.(*aresv1.AresJob)
	log := util.LoggerForJob(job)
	if !IsFinished(job.Status.Phase) && job.Spec.EnableGangScheduling() {
		minMember := *runPolicy.SchedulingPolicy.MinAvailable
		pgSpec := v1beta1.PodGroupSpec{
			MinMember: minMember,
		}
		podGroup, err := r.JobController.SyncPodGroup(job, pgSpec)
		if err != nil {
			log.Errorf("failed to sync podgroup: %v", err)
			return err
		}
		phase := podGroup.Status.Phase
		if len(phase) == 0 || phase == v1beta1.PodGroupPending {
			return NewReconcileError(fmt.Sprintf("pod group is [%v]", phase), WithRetryAfter(time.Second), DisableReRaise())
		}
		log.Infof("phase of podgroup: [%s]", podGroup.Status.Phase)
	}
	// 避免脏数据；例如expectedCreation = -3
	if err := r.ResetExpectations(job); err != nil {
		return fmt.Errorf("failed to reset expectations: %v", err)
	}
	return r.JobController.ReconcileJobs(j, replicas, jobStatus, runPolicy)
}
