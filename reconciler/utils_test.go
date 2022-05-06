package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSortPodsByTypeIndex(t *testing.T) {
	pods := []*corev1.Pod{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "ares-task-1-record-2-dev-worker-3",
				Labels: map[string]string{
					"ares.io/replica-index": "3",
					"ares.io/replica-type":  "worker",
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "ares-task-1-record-2-dev-worker-1",
				Labels: map[string]string{
					"ares.io/replica-index": "1",
					"ares.io/replica-type":  "worker",
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "ares-task-1-record-2-dev-worker-2",
				Labels: map[string]string{
					"ares.io/replica-index": "2",
					"ares.io/replica-type":  "worker",
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "ares-task-1-record-2-dev-launcher-10",
				Labels: map[string]string{
					"ares.io/replica-index": "10",
					"ares.io/replica-type":  "launcher",
				},
			},
		},
	}

	SortPodsByRoleAndIndex(pods)
	assert.Equal(t, "ares-task-1-record-2-dev-launcher-10", pods[0].Name)
	assert.Equal(t, "ares-task-1-record-2-dev-worker-1", pods[1].Name)
	assert.Equal(t, "ares-task-1-record-2-dev-worker-2", pods[2].Name)
	assert.Equal(t, "ares-task-1-record-2-dev-worker-3", pods[3].Name)
}
