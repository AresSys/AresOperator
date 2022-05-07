package mpi

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler"
)

func TestMPIFramework_SetDefault(t *testing.T) {
	f := &MPIFramework{}
	job := &aresv1.AresJob{
		Spec: aresv1.AresJobSpec{
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleLauncher: {},
			},
		},
	}
	f.SetDefault(job)
	job.Default()

	assert.Equal(t, commonv1.RestartPolicyNever, job.Spec.RoleSpecs[aresv1.RoleLauncher].RestartPolicy)
}

func TestMpiFramework_GetRolePhase(t *testing.T) {
	f := &MPIFramework{
		Reconciler: &reconciler.Reconciler{
			Log: zap.New(),
		},
	}
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	var replicas int32 = 3
	cases := []struct {
		spec      aresv1.RoleSpec
		lastPhase aresv1.RolePhase
		status    commonv1.ReplicaStatus
		phase     aresv1.RolePhase
	}{
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec: &commonv1.ReplicaSpec{
					Replicas:      &replicas,
					RestartPolicy: commonv1.RestartPolicyNever,
				},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhaseRunning,
			status:    commonv1.ReplicaStatus{Active: 1, Succeeded: 1, Failed: 0},
			phase:     aresv1.RolePhaseFailed,
		},
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhaseFailed,
			status:    commonv1.ReplicaStatus{Active: 1, Succeeded: 1, Failed: 0},
			phase:     aresv1.RolePhaseFailed,
		},
	}
	for idx, c := range cases {
		status := aresv1.AresJobStatus{
			JobStatus: &commonv1.JobStatus{
				ReplicaStatuses: map[commonv1.ReplicaType]*commonv1.ReplicaStatus{
					aresv1.RoleWorker: &c.status,
				},
			},
			RoleStatuses: map[commonv1.ReplicaType]*aresv1.RoleStatus{
				aresv1.RoleWorker: &aresv1.RoleStatus{
					Phase: c.lastPhase,
				},
			},
		}
		result := f.GetRolePhase(aresv1.RoleWorker, c.spec, status)
		assert.Equal(t, c.phase, result)
		t.Logf("case %d: expected=%v, result=%v", idx, c.phase, result)
	}
}
