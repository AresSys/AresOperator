package node

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// Conditions: 节点状态
type Conditions map[corev1.NodeConditionType]corev1.NodeCondition

// String: toString
func (conds Conditions) String() string {
	if cond, ok := conds[corev1.NodeReady]; ok {
		return fmt.Sprintf("%v(%v): %s", cond.Type, cond.Status, cond.Message)
	}
	ctypes := []string{}
	for ctype := range conds {
		ctypes = append(ctypes, string(ctype))
	}
	for _, cond := range conds {
		return fmt.Sprintf("%s(%v): %s", strings.Join(ctypes, "/"), cond.Status, cond.Message)
	}
	return fmt.Sprintf("%d unhealthy conditions", len(conds))
}

// Manager: NodeManager
type Manager struct {
	lock           sync.RWMutex
	unhealthyNodes map[string]Conditions
}

// NewManager: 新建Manager
func NewManager() *Manager {
	return &Manager{
		lock:           sync.RWMutex{},
		unhealthyNodes: map[string]Conditions{},
	}
}

// IsNodeDown: 是否NodeDown；返回是否以及原因
func (m *Manager) IsNodeDown(name string) (bool, string) {
	var conds Conditions
	m.lock.RLock()
	conds = m.unhealthyNodes[name]
	m.lock.RUnlock()
	if len(conds) == 0 {
		return false, ""
	}
	return true, conds.String()
}

// RemoveNode: 移除Conditions标记；返回是否存在并移除
func (m *Manager) RemoveNode(name string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, exists := m.unhealthyNodes[name]; exists {
		delete(m.unhealthyNodes, name)
		return true
	}
	return false
}

// SetNode: 设置Conditions标记；返回是否第一次标记
func (m *Manager) SetNode(name string, conds Conditions) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, exists := m.unhealthyNodes[name]
	m.unhealthyNodes[name] = conds
	return !exists
}

// HandleGeneric: 处理节点状态；返回是否变为Unhealthy
func (m *Manager) HandleGeneric(node *corev1.Node) bool {
	conds := Conditions{}
	for _, cond := range node.Status.Conditions {
		ctype := cond.Type
		if cond.Status == corev1.ConditionUnknown {
			conds[ctype] = cond
		}
	}
	turnDown := false
	if len(conds) > 0 {
		turnDown = m.SetNode(node.Name, conds)
		if turnDown {
			log.Infof("node down: %s, %s", node.Name, conds.String())
		}
	} else {
		if removed := m.RemoveNode(node.Name); removed {
			log.Infof("node recovered: %s", node.Name)
		}
	}
	return turnDown
}

func (m *Manager) HandleUpdate(old, node *corev1.Node) bool {
	return m.HandleGeneric(node)
}
