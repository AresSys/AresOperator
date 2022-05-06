package reconciler

import (
	"context"
	"testing"

	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(aresv1.AddToScheme(scheme))
	utilruntime.Must(scheduling.AddToScheme(scheme))
}

func TestPodGroupManager_updateScheduledByPodGroup(t *testing.T) {
	pg := &scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
		},
		Status: scheduling.PodGroupStatus{
			Conditions: []scheduling.PodGroupCondition{
				{
					Type:   scheduling.PodGroupScheduled,
					Status: corev1.ConditionTrue,
					Reason: "tasks in gang are ready to be scheduled",
				},
			},
		},
	}
	job := &aresv1.AresJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
		},
	}
	job.Default()
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(pg, job).
		Build()
	manager := NewPodGroupManager(cli)

	updated, err := manager.updateScheduledByPodGroup(job, pg)
	assert.NoError(t, err)
	assert.True(t, updated)
	cond := job.Status.Conditions[0]
	t.Logf("condition: %#v", cond)
	assert.Equal(t, aresv1.JobScheduled, cond.Type)
	assert.Equal(t, corev1.ConditionTrue, cond.Status)
}

func TestPodGroupManager_checkEviction(t *testing.T) {
	pg := &scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
		},
		Status: scheduling.PodGroupStatus{
			Conditions: []scheduling.PodGroupCondition{
				{
					Type:    scheduling.PodGroupConditionType(aresv1.Evicted),
					Status:  corev1.ConditionTrue,
					Reason:  aresv1.Preempted,
					Message: "preempted podgroup",
				},
			},
		},
	}
	job := &aresv1.AresJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
		},
	}
	job.Default()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test-pod",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: aresv1.GroupVersion.String(),
				Kind:       aresv1.GroupVersionKind.Kind,
				Name:       "job-test",
			}},
		},
		Spec: corev1.PodSpec{},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{{
				Type:    corev1.PodReady,
				Status:  corev1.ConditionFalse,
				Reason:  aresv1.Preempted,
				Message: "preempted pod",
			}},
		},
	}
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(job, pod, &scheduling.PodGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "job-test",
				Namespace: "default",
			},
		}).
		Build()
	manager := NewPodGroupManager(cli)

	// evicted determined by pod
	pgRunning := pg.DeepCopy()
	pgRunning.Status = scheduling.PodGroupStatus{}
	evicted := manager.checkEviction(job, pgRunning)
	assert.Equal(t, 0, len(job.Status.Conditions))
	assert.False(t, *evicted)

	// evicted determined by podgroup
	err := cli.Update(context.Background(), pg)
	assert.NoError(t, err)
	evicted = manager.checkEviction(job, pg)
	assert.True(t, *evicted)
	cond := job.Status.Conditions[0]
	t.Logf("condition: %#v", cond)
	assert.Equal(t, commonv1.JobFailed, cond.Type)
	assert.Equal(t, aresv1.Preempted, cond.Reason)
}
