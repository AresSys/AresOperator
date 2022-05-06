package reconciler

import (
	"fmt"

	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"

	aresv1 "ares-operator/api/v1"
)

func (r *Reconciler) ExpectationsFuncApply(job *aresv1.AresJob, f func(key string) error) error {
	key, err := common.KeyFunc(job)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for job object %v: %v", job, err))
		return err
	}
	for rtype := range job.Spec.RoleSpecs {
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		if err := f(expectationPodsKey); err != nil {
			return err
		}

		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		if err := f(expectationServicesKey); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) ResetExpectations(job *aresv1.AresJob) error {
	return r.ExpectationsFuncApply(job, func(key string) error {
		// Clear the expectations
		return r.Expectations.SetExpectations(key, 0, 0)
	})
}

func (r *Reconciler) SatisfiedExpectations(job *aresv1.AresJob) bool {
	err := r.ExpectationsFuncApply(job, func(key string) error {
		// Check the expectations
		if satisfied := r.Expectations.SatisfiedExpectations(key); !satisfied {
			return fmt.Errorf("expectation is not satisfied: %s", key)
		}
		return nil
	})
	return err == nil
}

func (r *Reconciler) DeleteExpectations(job *aresv1.AresJob) {
	r.ExpectationsFuncApply(job, func(key string) error {
		// Delete the expectations
		r.Expectations.DeleteExpectations(key)
		return nil
	})
}

func (r *Reconciler) AddObject(job *aresv1.AresJob, obj metav1.Object) {
	key, err := common.KeyFunc(job)
	if err != nil {
		return
	}
	rtype := GetReplicaType(obj.GetLabels())
	if len(rtype) == 0 {
		return
	}

	var expectKey string
	if _, ok := obj.(*corev1.Pod); ok {
		expectKey = expectation.GenExpectationPodsKey(key, rtype)
	}
	if _, ok := obj.(*corev1.Service); ok {
		expectKey = expectation.GenExpectationServicesKey(key, rtype)
	}
	util.LoggerForJob(job).Infof("creation observed: name=%s, key=%s", obj.GetName(), expectKey)
	r.Expectations.CreationObserved(expectKey)
}

func (r *Reconciler) DeleteObject(job *aresv1.AresJob, obj metav1.Object) {
	key, err := common.KeyFunc(job)
	if err != nil {
		return
	}
	rtype := GetReplicaType(obj.GetLabels())
	if len(rtype) == 0 {
		return
	}

	var expectKey string
	if _, ok := obj.(*corev1.Pod); ok {
		expectKey = expectation.GenExpectationPodsKey(key, rtype)
	}
	if _, ok := obj.(*corev1.Service); ok {
		expectKey = expectation.GenExpectationServicesKey(key, rtype)
	}
	util.LoggerForJob(job).Infof("deletion observed: name=%s, key=%s", obj.GetName(), expectKey)
	r.Expectations.DeletionObserved(expectKey)
}
