package cache

import (
	"fmt"
	"strings"

	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

type RoleCache struct {
	Phase   aresv1.RolePhase `json:"phase"`
	Running bool             `json:"running"`
	IPs     []string         `json:"ips,omitempty"`
}

type RoleCacheOptionFunc func(c *RoleCache)

func WithIPs(ips []string) RoleCacheOptionFunc {
	return func(c *RoleCache) {
		c.IPs = ips
	}
}

// NewRoleCache: 创建RoleCache
func NewRoleCache(phase aresv1.RolePhase, opts ...RoleCacheOptionFunc) *RoleCache {
	c := &RoleCache{
		Phase:   phase,
		Running: phase == aresv1.RolePhaseRunning,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type JobCache map[commonv1.ReplicaType]*RoleCache

func NamespacedNameToKey(ns, name string) string {
	return fmt.Sprintf("%s.%s", ns, name)
}

func KeyToNamespacedName(key string) (string, string, error) {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("failed to split '%s' to 2 parts: %v", key, parts)
	}
	return parts[0], parts[1], nil
}
