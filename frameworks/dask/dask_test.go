package dask

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler"
)

func TestDependencyRole(t *testing.T) {
	dask := &Dask{}

	job := &aresv1.AresJob{
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleScheduler: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleMaster: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleWorker: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleLauncher: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
			},
		},
	}
	dask.SetDefault(job)
	job.Default()
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleScheduler}, job.Spec.RoleSpecs[aresv1.RoleMaster].Dependence.RoleNames)
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleMaster}, job.Spec.RoleSpecs[aresv1.RoleWorker].Dependence.RoleNames)
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleWorker}, job.Spec.RoleSpecs[aresv1.RoleLauncher].Dependence.RoleNames)

	job2 := &aresv1.AresJob{
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleScheduler: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleMaster: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleLauncher: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
			},
		},
	}
	dask.SetDefault(job2)
	job2.Default()
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleScheduler}, job2.Spec.RoleSpecs[aresv1.RoleMaster].Dependence.RoleNames)
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleMaster}, job2.Spec.RoleSpecs[aresv1.RoleLauncher].Dependence.RoleNames)

	job3 := &aresv1.AresJob{
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleScheduler: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},

				aresv1.RoleLauncher: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
			},
		},
	}
	dask.SetDefault(job3)
	job3.Default()
	assert.Equal(t, []commonv1.ReplicaType{aresv1.RoleScheduler}, job3.Spec.RoleSpecs[aresv1.RoleLauncher].Dependence.RoleNames)
}

func TestSetDefaultValues(t *testing.T) {
	dask := &Dask{}

	job := &aresv1.AresJob{
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleScheduler: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleMaster: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleWorker: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleLauncher: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
			},
		},
	}
	dask.SetDefault(job)
	job.Default()

	//check failed condition
	assert.Equal(t,
		[][]commonv1.ReplicaType{{aresv1.RoleLauncher}, {aresv1.RoleScheduler}, {aresv1.RoleMaster}, {aresv1.RoleWorker}},
		job.Spec.JobPhasePolicy.Failed)

	//check role condation
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	assert.Equal(t, &any, job.Spec.RoleSpecs[aresv1.RoleScheduler].RolePhasePolicy.Failed)
	assert.Equal(t, &all, job.Spec.RoleSpecs[aresv1.RoleScheduler].RolePhasePolicy.Succeeded)

	assert.Equal(t, &any, job.Spec.RoleSpecs[aresv1.RoleLauncher].RolePhasePolicy.Failed)
	assert.Equal(t, &all, job.Spec.RoleSpecs[aresv1.RoleLauncher].RolePhasePolicy.Succeeded)

	assert.Equal(t, &any, job.Spec.RoleSpecs[aresv1.RoleMaster].RolePhasePolicy.Failed)
	assert.Equal(t, &all, job.Spec.RoleSpecs[aresv1.RoleMaster].RolePhasePolicy.Succeeded)

	assert.Equal(t, &any, job.Spec.RoleSpecs[aresv1.RoleWorker].RolePhasePolicy.Failed)
	assert.Equal(t, &all, job.Spec.RoleSpecs[aresv1.RoleWorker].RolePhasePolicy.Succeeded)

}

func TestEnvironments(t *testing.T) {
	dask := &Dask{}

	job := &aresv1.AresJob{
		ObjectMeta: v1.ObjectMeta{
			Name:      "dummy-dask-name",
			Namespace: "dummy-namespace",
		},
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleScheduler: {
					ReplicaSpec: &commonv1.ReplicaSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name: dask.GetDefaultContainerName(),
									Ports: []corev1.ContainerPort{{
										Name:          "",
										ContainerPort: 123,
									}},
								}},
							},
						},
					},
				},
				aresv1.RoleMaster: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleWorker: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
				aresv1.RoleLauncher: {
					ReplicaSpec: &commonv1.ReplicaSpec{},
				},
			},
		},
	}
	podTemplate := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: dask.GetDefaultContainerName(),
				Env:  []corev1.EnvVar{},
			}},
		},
	}
	dask.SetDefault(job)
	err := dask.SetClusterSpec(job, podTemplate, string(aresv1.RoleLauncher), "0")
	assert.Nil(t, err)

	defaultContainer := podTemplate.Spec.Containers[0]
	var schedulerPortEnv, schedulerHost corev1.EnvVar

	for _, env := range defaultContainer.Env {
		if env.Name == "SchedulerPort" {
			schedulerPortEnv = env

		}
		if env.Name == "SchedulerAddress" {
			schedulerHost = env
		}
	}
	assert.NotNil(t, schedulerPortEnv)
	assert.NotNil(t, schedulerHost)
	assert.Equal(t, "123", schedulerPortEnv.Value)
	assert.Equal(t, reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleScheduler), schedulerHost.Value)
}
