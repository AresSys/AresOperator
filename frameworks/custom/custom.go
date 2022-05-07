package custom

import (
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
)

var _ fcommon.Framework = &Custom{}

var CustomBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &Custom{Reconciler: r}
}

type Custom struct {
	*reconciler.Reconciler
}

func (c *Custom) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkCustom
}

func (c *Custom) SetDefault(job *aresv1.AresJob) {
	// job phase policy
	setCommonJobPhasePolicy(job)
	// role phase policy
	setCommonRolePhasePolicy(job)
}

func (c *Custom) Validate(job *aresv1.AresJob) error {
	return nil
}

func setCommonJobPhasePolicy(job *aresv1.AresJob) {
	if job.Spec.JobPhasePolicy != nil {
		return
	}

	succeeded := [][]commonv1.ReplicaType{{}}
	failed := [][]commonv1.ReplicaType{}
	for role := range job.Spec.RoleSpecs {
		succeeded[0] = append(succeeded[0], role)
		failed = append(failed, []commonv1.ReplicaType{role})
	}
	job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
		Succeeded: succeeded,
		Failed:    failed,
	}
}

func setCommonRolePhasePolicy(job *aresv1.AresJob) {
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	for _, spec := range job.Spec.RoleSpecs {
		if spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
		if spec.RestartPolicy == commonv1.RestartPolicyNever ||
			spec.RestartPolicy == commonv1.RestartPolicyExitCode {
			if spec.RolePhasePolicy.Failed == nil {
				spec.RolePhasePolicy.Failed = &any
			}
		}
	}
}

func (c *Custom) SetClusterSpec(j interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	job := j.(*aresv1.AresJob)
	// add env
	for i := range podTemplate.Spec.Containers {
		podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, []corev1.EnvVar{
			{Name: "MODULE", Value: rtype},
			{Name: "INDEX", Value: index},
		}...)
		for role := range job.Spec.RoleSpecs {
			podTemplate.Spec.Containers[i].Env = append(podTemplate.Spec.Containers[i].Env, []corev1.EnvVar{
				{Name: string(role), Value: strings.Join(reconciler.GetRolesDomainNames(job, role), ",")},
			}...)
		}
	}
	return nil
}
