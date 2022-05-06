package reconciler

import (
	"testing"

	"github.com/bombsimon/logrusr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler/node"
	"ares-operator/utils/fake"
)

func TestInitializeReplicaStatuses(t *testing.T) {
	job := &aresv1.AresJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", UID: "uuid"},
		Status: aresv1.AresJobStatus{
			RoleStatuses: map[commonv1.ReplicaType]*aresv1.RoleStatus{
				aresv1.RoleWorker: {
					ReplicaStatus: aresv1.ReplicaStatus{
						ReplicaStatus: &commonv1.ReplicaStatus{
							Active: 1,
						},
						Unknown: 1,
					},
				},
			},
		},
	}
	initializeReplicaStatuses(&job.Status, aresv1.RoleWorker)
	unknown := job.Status.RoleStatuses[aresv1.RoleWorker].Unknown
	active := job.Status.RoleStatuses[aresv1.RoleWorker].Active
	t.Logf("Unknown: %d; Active: %d", unknown, active)
	assert.Equal(t, unknown, int32(0))
	assert.Equal(t, active, int32(0))
}

func TestReconciler_UpdateRolePhase(t *testing.T) {
	log := logrusr.NewLogger(logrus.StandardLogger())
	r := Reconciler{
		JobController: common.JobController{
			Recorder: fake.NewFakeRecorder(log),
		},
		Log:         log,
		NodeManager: node.NewManager(),
	}
	var replicas int32 = 1
	any := aresv1.RolePhaseConditionAnyPod
	job := &aresv1.AresJob{
		ObjectMeta: metav1.ObjectMeta{Name: "test-job", UID: "uuid"},
		Spec: aresv1.AresJobSpec{
			FrameworkType: "test-framework",
			RoleSpecs: map[commonv1.ReplicaType]*aresv1.RoleSpec{
				aresv1.RoleWorker: {
					ReplicaSpec: &commonv1.ReplicaSpec{
						Replicas: &replicas,
					},
					RolePhasePolicy: aresv1.RolePhasePolicy{
						Failed:           &any,
						NodeDownAsFailed: true,
					},
				},
			},
		},
		Status: aresv1.AresJobStatus{
			JobStatus: &commonv1.JobStatus{
				ReplicaStatuses: map[commonv1.ReplicaType]*commonv1.ReplicaStatus{
					aresv1.RoleWorker: {},
				},
			},
		},
	}
	pod := &corev1.Pod{}
	pod.Labels = map[string]string{commonv1.ReplicaIndexLabel: "0"}
	pod.Spec.NodeName = "fake-node"
	pod.Status.Phase = corev1.PodRunning
	r.NodeManager.SetNode(pod.Spec.NodeName, node.Conditions{corev1.NodeReady: corev1.NodeCondition{}})
	r.UpdateRolePhase(job, aresv1.RoleWorker, []*corev1.Pod{pod})
}

func TestGetJobPhase(t *testing.T) {
	policy := aresv1.JobPhasePolicy{
		Failed:    [][]commonv1.ReplicaType{{aresv1.RolePS}, {aresv1.RoleHub}, {aresv1.RoleTrain}},
		Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleTrain}},
	}
	cases := []struct {
		policy    aresv1.JobPhasePolicy
		status    map[commonv1.ReplicaType]*aresv1.RoleStatus
		lastPhase aresv1.JobPhase
		phase     aresv1.JobPhase
	}{
		{policy: policy, status: map[commonv1.ReplicaType]*aresv1.RoleStatus{
			aresv1.RolePS:    {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleHub:   {Phase: aresv1.RolePhaseFailed},
			aresv1.RoleTrain: {Phase: aresv1.RolePhaseSucceeded},
		}, lastPhase: aresv1.JobPhaseRunning, phase: aresv1.JobPhaseFailed},
		{policy: policy, status: map[commonv1.ReplicaType]*aresv1.RoleStatus{
			aresv1.RolePS:    {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleHub:   {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleTrain: {Phase: aresv1.RolePhaseSucceeded},
		}, lastPhase: aresv1.JobPhaseRunning, phase: aresv1.JobPhaseSucceeded},
		{policy: policy, status: map[commonv1.ReplicaType]*aresv1.RoleStatus{
			aresv1.RolePS:    {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleHub:   {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleTrain: {Phase: aresv1.RolePhaseRunning},
		}, lastPhase: aresv1.JobPhasePending, phase: aresv1.JobPhaseRunning},
		{policy: policy, status: map[commonv1.ReplicaType]*aresv1.RoleStatus{
			aresv1.RolePS:    {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleHub:   {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleTrain: {Phase: aresv1.RolePhasePending},
		}, lastPhase: aresv1.JobPhasePending, phase: aresv1.JobPhaseStarting},
		{policy: policy, status: map[commonv1.ReplicaType]*aresv1.RoleStatus{
			aresv1.RolePS:    {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleHub:   {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleTrain: {Phase: aresv1.RolePhaseRunning},
			aresv1.RoleEtcd:  {Phase: aresv1.RolePhaseFailed},
		}, lastPhase: aresv1.JobPhaseRunning, phase: aresv1.JobPhaseRunning},
	}
	stub := &Reconciler{}
	for idx, c := range cases {
		result := stub.GetJobPhase(c.lastPhase, c.policy, c.status)
		assert.Equal(t, c.phase, result)
		t.Logf("case %d: expected=%v, result=%v", idx, c.phase, result)
	}
}

func TestGetRolePhase(t *testing.T) {
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	var replicas int32 = 3
	cases := []struct {
		policy    aresv1.RolePhasePolicy
		spec      aresv1.RoleSpec
		lastPhase aresv1.RolePhase
		status    commonv1.ReplicaStatus
		phase     aresv1.RolePhase
	}{
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhasePending,
			status:    commonv1.ReplicaStatus{Active: 0, Succeeded: 0, Failed: 1},
			phase:     aresv1.RolePhaseFailed,
		},
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhasePending,
			status:    commonv1.ReplicaStatus{Active: 0, Succeeded: 3, Failed: 0},
			phase:     aresv1.RolePhaseSucceeded,
		},
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhasePending,
			status:    commonv1.ReplicaStatus{Active: 3, Succeeded: 0, Failed: 0},
			phase:     aresv1.RolePhaseRunning,
		},
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhasePending,
			status:    commonv1.ReplicaStatus{Active: 1, Succeeded: 0, Failed: 0},
			phase:     aresv1.RolePhaseStarting,
		},
		{
			spec: aresv1.RoleSpec{
				ReplicaSpec:     &commonv1.ReplicaSpec{Replicas: &replicas},
				RolePhasePolicy: aresv1.RolePhasePolicy{Failed: &any, Succeeded: &all},
			},
			lastPhase: aresv1.RolePhaseRunning,
			status:    commonv1.ReplicaStatus{Active: 2, Succeeded: 1, Failed: 0},
			phase:     aresv1.RolePhaseRunning,
		},
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
					ReplicaStatus: aresv1.ReplicaStatus{
						ReplicaStatus: &c.status,
					},
				},
			},
		}
		stub := &Reconciler{}
		result := stub.GetRolePhase(aresv1.RoleWorker, c.spec, status)
		assert.Equal(t, c.phase, result)
		t.Logf("case %d: expected=%v, result=%v", idx, c.phase, result)
	}
}
