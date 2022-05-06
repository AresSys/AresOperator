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

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/util"
	logger "github.com/kubeflow/common/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/apis/pkg/client/clientset/versioned"

	aresv1 "ares-operator/api/v1"
	"ares-operator/frameworks"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
	arescontrol "ares-operator/reconciler/control"
	"ares-operator/utils/fake"
)

// AresJobReconciler reconciles a AresJob object
type AresJobReconciler struct {
	Reconciler   *reconciler.Reconciler
	Frameworks   map[aresv1.FrameworkType]fcommon.Framework
	CleanUpQueue workqueue.RateLimitingInterface
}

//+kubebuilder:rbac:groups=ares.io,resources=aresjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ares.io,resources=aresjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ares.io,resources=aresjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=scheduling.volcano.sh,resources=podgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AresJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *AresJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	aresjob := &aresv1.AresJob{}
	err := r.Reconciler.Get(ctx, req.NamespacedName, aresjob)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// requeue
		return ctrl.Result{Requeue: true}, err
	}
	job := aresjob.DeepCopy() // Note(important): do not update struct in informer cache
	log := logger.LoggerForJob(job)
	log.Infof(" >>> reconcile start")
	defer func() {
		log.Infof(" <<< reconcile end")
	}()

	if job.DeletionTimestamp != nil {
		log.Info("reconcile skipped, job has been deleted.")
		if err := r.finalize(job); err != nil {
			log.Errorf("failed to finalize: %v", err)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, nil
	}
	if reconciler.IsInvalidDefinition(job.Status) {
		return ctrl.Result{}, nil
	}

	framework := r.getFramework(job.Spec.FrameworkType)
	if framework == nil {
		msg := fmt.Sprintf("unknown framework: %s", job.Spec.FrameworkType)
		log.Warning(msg)
		r.setInvalidDefinition(job, msg)
		return ctrl.Result{}, nil
	}

	// Set default priorities for job
	if updated, err := r.setDefault(aresjob, job, framework); err != nil {
		return ctrl.Result{Requeue: true}, err
	} else {
		if updated {
			return ctrl.Result{}, nil
		}
	}
	// Validate definition
	if err := r.validate(job, framework); err != nil {
		// do not need requeue
		log.Errorf("failed to validate: %v", err)
		return ctrl.Result{}, nil
	}

	needSync := r.Reconciler.SatisfiedExpectations(job)
	if !needSync {
		log.Info("reconcile skipped, expectation is not satisfied")
		return ctrl.Result{}, nil
	}
	needReconcile := true
	if action, err := r.Reconciler.SyncFromPodGroup(job, r.Reconciler); err != nil {
		return ctrl.Result{Requeue: true}, err
	} else if action == reconciler.ActionBreak {
		return ctrl.Result{}, nil
	} else if action == reconciler.ActionSkip {
		log.Info("reconcile skipped, job has been evicted")
		needReconcile = false
	}

	// Use framework to reconcile the job related pod and service
	if needReconcile {
		err = framework.ReconcileJobs(job, job.Spec.GetReplicaSpecs(), *job.Status.JobStatus, &job.Spec.RunPolicy)
	}
	if err != nil {
		log.WithError(err).Error("reconcile jobs error")
		if errors.IsInvalid(err) { // 例如Pod定义非法
			r.setInvalidDefinition(job, fmt.Sprintf("failed to reconcile: %v", err))
			return ctrl.Result{}, nil
		} else if e, ok := reconciler.IsReconcileError(err); ok {
			return ctrl.Result{RequeueAfter: e.RetryAfter}, e.Err()
		}
		return ctrl.Result{Requeue: true}, err
	}

	if err := r.Reconciler.SyncFromJob(job); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	if err := r.HandleRolePhases(aresjob, job); err != nil {
		log.WithError(err).Error("handle role phases error")
		return ctrl.Result{Requeue: true}, err
	}
	if reconciler.IsFinished(job.Status.Phase) {
		r.OnJobFinished(job)
	}
	return ctrl.Result{}, nil
}

func (r *AresJobReconciler) setDefault(old, job *aresv1.AresJob, framework fcommon.Framework) (bool, error) {
	scheme.Scheme.Default(job)
	framework.SetDefault(job)
	job.Default()
	if reflect.DeepEqual(old.Spec, job.Spec) {
		return false, nil
	}
	log := util.LoggerForJob(job)
	if err := r.Reconciler.Update(context.Background(), job); err != nil {
		log.Errorf("failed to update AresJob spec in the API server: %v", err)
		return false, err
	}
	log.Infof("succeeded to update default properties in the API server: %s", job.Json())
	return true, nil
}

func (r *AresJobReconciler) validate(job *aresv1.AresJob, framework fcommon.Framework) error {
	if err := job.ValidateCreate(); err != nil {
		r.setInvalidDefinition(job, fmt.Sprintf("common validation failed: %v", err))
		return err
	}
	if err := framework.Validate(job); err != nil {
		r.setInvalidDefinition(job, fmt.Sprintf("framework validation failed: %v", err))
		return err
	}
	return nil
}

func (r *AresJobReconciler) setInvalidDefinition(job *aresv1.AresJob, msg string) error {
	reconciler.SetInvalidDefinition(&job.Status, msg)
	return r.Reconciler.Status().Update(context.Background(), job)
}

func (r *AresJobReconciler) getFramework(name aresv1.FrameworkType) fcommon.Framework {
	return r.Frameworks[name]
}

// SetupWithManager sets up the controller with the Manager.
func (r *AresJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.setupUnderlyingJobController(mgr)
	r.setupFrameworks(mgr)
	r.setupCleanUpQueue()
	if err := r.setupIndexer(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&aresv1.AresJob{}).
		Owns(&v1.Pod{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: r.AddObject,
			UpdateFunc: r.UpdateObject,
			DeleteFunc: r.DeleteObject,
		})).
		Owns(&v1.Service{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: r.AddObject,
			DeleteFunc: r.DeleteObject,
		})).
		Owns(&scheduling.PodGroup{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: r.UpdateObject,
		})).
		Watches(&source.Kind{Type: &v1.Node{}}, &NodeEventHandler{
			manager: r.Reconciler.NodeManager,
			r:       r.Reconciler,
		}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

func (r *AresJobReconciler) setupUnderlyingJobController(mgr ctrl.Manager) {
	recorder := mgr.GetEventRecorderFor(r.ControllerName())

	restConfig := mgr.GetConfig()
	restConfig.QPS = r.Reconciler.Config.GetRestConfig().QPS
	restConfig.Burst = r.Reconciler.Config.GetRestConfig().Burst
	kubeClient := kubernetes.NewForConfigOrDie(restConfig)
	r.Reconciler.JobController = common.JobController{
		Expectations: expectation.NewControllerExpectations(),
		//是否使用gangScheduling由我们控制
		Config:           common.JobControllerConfiguration{EnableGangScheduling: false},
		WorkQueue:        &fake.FakeWorkQueue{},
		Recorder:         recorder,
		KubeClientSet:    kubernetes.NewForConfigOrDie(restConfig),
		VolcanoClientSet: versioned.NewForConfigOrDie(restConfig),
		PodControl: arescontrol.RealPodControl{
			RealPodControl: &control.RealPodControl{KubeClient: kubeClient, Recorder: recorder},
		},
		ServiceControl: control.RealServiceControl{KubeClient: kubeClient, Recorder: recorder},
	}
}

func (r *AresJobReconciler) setupFrameworks(mgr ctrl.Manager) {
	r.Frameworks = map[aresv1.FrameworkType]fcommon.Framework{}
	for _, builder := range frameworks.Factory.Builders {
		f := builder(r.Reconciler.Copy(), mgr)
		f.SetController(f)
		r.Frameworks[f.GetName()] = f
	}
}

// setupCleanUpQueue: 设置清理队列
func (r *AresJobReconciler) setupCleanUpQueue() {
	r.CleanUpQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	go wait.Forever(func() {
		for r.cleanupNextItem() {
		}
	}, time.Second)
}

func (r *AresJobReconciler) setupIndexer(mgr ctrl.Manager) error {
	// add job owner key(https://book.kubebuilder.io/cronjob-tutorial/controller-implementation.html setup section)
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, aresv1.JobOwnerKey, IndexObjectOwner); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, aresv1.ReplicaOwnerKey, IndexReplicaOwner); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Service{}, aresv1.JobOwnerKey, IndexObjectOwner); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Service{}, aresv1.ReplicaOwnerKey, IndexReplicaOwner); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, aresv1.NodeNameKey, func(rawObj client.Object) []string {
		pod := rawObj.(*v1.Pod)
		owner := metav1.GetControllerOf(pod)
		if !aresv1.IsAresJob(owner) {
			return nil
		}
		if len(pod.Spec.NodeName) > 0 {
			return []string{pod.Spec.NodeName}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
