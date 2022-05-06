/*
Copyright 2021 KML Ares-Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package observer

import (
	"context"
	"fmt"

	aresv1 "ares-operator/api/v1"
	"ares-operator/cache"
	"ares-operator/controllers"
	"ares-operator/observer/server"
	"ares-operator/reconciler"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

var (
	ServerLog = ctrl.Log.WithName("http-server")
	observer  *AresJobObserver
)

type AresJobObserver struct {
	client.Client
}

func NewObserver(c client.Client) *AresJobObserver {
	return &AresJobObserver{
		Client: c,
	}
}

func (r *AresJobObserver) Register() {
	observer = r
}

func (r *AresJobObserver) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *AresJobObserver) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.setupIndexer(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&aresv1.AresJob{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 0,
		}).
		Complete(r)
}

func (r *AresJobObserver) setupIndexer(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, aresv1.JobOwnerKey, controllers.IndexObjectOwner); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, aresv1.ReplicaOwnerKey, controllers.IndexReplicaOwner); err != nil {
		return err
	}
	return nil
}

// GetJob: 根据名称获取AresJob
func (r *AresJobObserver) GetJob(ns, name string) (*aresv1.AresJob, error) {
	job := &aresv1.AresJob{}
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, job); err != nil {
		return nil, err
	}
	return job, nil
}

// GetPodsForJob: 获取指定任务的实例列表
func (r *AresJobObserver) GetPodsForJob(job *aresv1.AresJob) ([]*corev1.Pod, error) {
	return reconciler.GetPodsForJob(r.Client, job)
}

// GetJobCache: 获取JobCache
func GetJobCache(ns, name string) (cache.JobCache, error) {
	job, err := observer.GetJob(ns, name)
	if err != nil {
		code := UnknownError
		if errors.IsNotFound(err) {
			code = NotFound
		}
		return nil, server.NewServiceError(code, "failed to get aresjob for '%s/%s': %v", ns, name, err)
	}
	pods, err := observer.GetPodsForJob(job)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods for '%s/%s': %v", ns, name, err)
	}
	jobCache := cache.JobCache{}
	for role, status := range job.Status.RoleStatuses {
		roleCache := reconciler.BuildRoleCache(job, role, status.Phase, pods)
		jobCache[role] = roleCache
	}
	return jobCache, nil
}
