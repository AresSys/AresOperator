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

package v1

import (
	"fmt"

	"ares-operator/metadata"
	"ares-operator/utils"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var aresjoblog = logf.Log.WithName("aresjob-resource")
var defaultCleanPodPolicy = commonv1.CleanPodPolicyNone

func (r *AresJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-ares-io-v1-aresjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=ares.io,resources=aresjobs,verbs=create;update,versions=v1,name=maresjob.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &AresJob{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AresJob) Default() {
	// metadata
	if r.Annotations == nil {
		r.Annotations = map[string]string{}
	}
	if r.Spec.HasDependence() && !controllerutil.ContainsFinalizer(r, AresJobFinalizer) {
		controllerutil.AddFinalizer(r, AresJobFinalizer)
	}
	if r.Labels == nil {
		r.Labels = map[string]string{}
	}
	for key, value := range r.Annotations {
		if metadata.IsJobLevelKey(key) {
			r.Labels[key] = value
		}
	}
	// spec
	if r.Spec.RunPolicy.CleanPodPolicy == nil {
		r.Spec.RunPolicy.CleanPodPolicy = &defaultCleanPodPolicy
	}
	for _, spec := range r.Spec.RoleSpecs {
		if spec.ReplicaSpec == nil {
			spec.ReplicaSpec = &commonv1.ReplicaSpec{}
		}
		if len(spec.Controller) == 0 {
			spec.Controller = ControllerPod
		}
		if len(spec.RestartPolicy) == 0 {
			spec.RestartPolicy = commonv1.RestartPolicyNever
		}
		if spec.RestartPolicy == commonv1.RestartPolicyNever {
			if spec.RolePhasePolicy.Failed == nil {
				any := RolePhaseConditionAnyPod
				spec.RolePhasePolicy.Failed = &any
			}
			if spec.RolePhasePolicy.Succeeded == nil {
				all := RolePhaseConditionAllPods
				spec.RolePhasePolicy.Succeeded = &all
			}
		}
		if spec.Replicas == nil {
			one := int32(1)
			spec.Replicas = &one
		}
	}
	// status
	if r.Status.JobStatus == nil {
		r.Status.JobStatus = &commonv1.JobStatus{}
	}
	if r.Status.RoleStatuses == nil {
		r.Status.RoleStatuses = map[commonv1.ReplicaType]*RoleStatus{}
	}
	for role := range r.Spec.RoleSpecs {
		if r.Status.RoleStatuses[role] == nil {
			r.Status.RoleStatuses[role] = &RoleStatus{
				Phase: RolePhasePending,
			}
		}
	}
	if len(r.Status.Phase) == 0 {
		r.Status.Phase = JobPhasePending
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-ares-io-v1-aresjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=ares.io,resources=aresjobs,verbs=create;update,versions=v1,name=varesjob.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &AresJob{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AresJob) ValidateCreate() error {
	if len(r.Spec.RoleSpecs) == 0 {
		return fmt.Errorf("spec.roleSpecs is required")
	}
	for role, spec := range r.Spec.RoleSpecs {
		if err := utils.MatchDNS1123(string(role)); err != nil {
			return fmt.Errorf("illegal name for role: %v", err)
		}
		if spec.Replicas == nil || *spec.Replicas <= 0 {
			return fmt.Errorf("replicas for role %s must be positive: %d", role, *spec.Replicas)
		}
		if len(spec.Template.Spec.Containers) == 0 {
			return fmt.Errorf("too few container in role: %v", role)
		}
		if spec.Dependence != nil && spec.Dependence.MPIHostFile == nil && len(spec.Dependence.RoleNames) == 0 {
			return fmt.Errorf("`spec.Dependence.RoleNames` is required when `MPIHostFile` is not provided: %v", role)
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AresJob) ValidateUpdate(old runtime.Object) error {
	aresjoblog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AresJob) ValidateDelete() error {
	aresjoblog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
