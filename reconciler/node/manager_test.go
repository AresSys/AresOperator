package node

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestManager(t *testing.T) {
	m := NewManager()
	conds := Conditions{corev1.NodeReady: corev1.NodeCondition{
		Type:    corev1.NodeReady,
		Status:  corev1.ConditionUnknown,
		Reason:  "NodeStatusUnknown",
		Message: "Kubelet stopped posting node status.",
	}}
	nodeName := "bjpg-g271.yz02"
	turnDown := m.SetNode(nodeName, conds)
	assert.True(t, turnDown)

	nodeDown, reason := m.IsNodeDown(nodeName)
	assert.True(t, nodeDown)
	t.Logf("is node down? %v, %s", nodeDown, reason)

	updated := m.RemoveNode(nodeName)
	assert.True(t, updated)

	updated = m.RemoveNode(nodeName)
	assert.False(t, updated)
}

func TestManager_HandleGeneric(t *testing.T) {
	m := NewManager()
	nodeName := "bjpg-g271.yz02"

	unhealthy := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
			{
				Type:    corev1.NodeReady,
				Status:  corev1.ConditionUnknown,
				Reason:  "NodeStatusUnknown",
				Message: "Kubelet stopped posting node status.",
			},
		}},
	}
	healthy := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{
			{
				Type:    corev1.NodeReady,
				Status:  corev1.ConditionTrue,
				Reason:  "KubeletReady",
				Message: "kubelet is posting ready status",
			},
		}},
	}
	turnDown := m.HandleGeneric(unhealthy)
	assert.True(t, turnDown)
	nodeDown, reason := m.IsNodeDown(nodeName)
	assert.True(t, nodeDown)
	t.Logf("is node down? %v, %s", nodeDown, reason)

	turnDown = m.HandleGeneric(healthy)
	assert.False(t, turnDown)
	nodeDown, reason = m.IsNodeDown(nodeName)
	assert.False(t, nodeDown)
	t.Logf("is node down? %v, %s", nodeDown, reason)
}
