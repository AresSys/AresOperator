package cache

import (
	"fmt"

	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
)

var _ Interface = &MemoryCache{}

type MemoryCache struct {
	cache map[string]map[commonv1.ReplicaType]*RoleCache
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: map[string]map[commonv1.ReplicaType]*RoleCache{},
	}
}

func (c *MemoryCache) Close() {
	c.cache = nil
}

// getSubCache: 获取SubCache
func (c *MemoryCache) getSubCache(key string) (map[commonv1.ReplicaType]*RoleCache, error) {
	if c.cache == nil {
		return nil, fmt.Errorf("MemoryCache has been closed")
	}
	if c.cache[key] == nil {
		c.cache[key] = map[commonv1.ReplicaType]*RoleCache{}
	}
	return c.cache[key], nil
}

// SetRoleCache: 设置指定角色的缓存信息
func (c *MemoryCache) SetRoleCache(jobKey string, roleName commonv1.ReplicaType, data *RoleCache) error {
	roleKey := aresv1.MetaReplicaKeyFormat(jobKey, roleName)
	sc, err := c.getSubCache(jobKey)
	if err != nil {
		return err
	}
	sc[roleName] = data
	log := util.LoggerForKey(roleKey)
	log.Infof("succeeded to set role cache: %+v", data)
	return nil
}

// GetRoleCache: 获取指定角色的缓存信息
func (c *MemoryCache) GetRoleCache(jobKey string, roleName commonv1.ReplicaType) (*RoleCache, error) {
	roleKey := aresv1.MetaReplicaKeyFormat(jobKey, roleName)
	sc, err := c.getSubCache(jobKey)
	if err != nil {
		return nil, err
	}
	roleCache := sc[roleName]
	log := util.LoggerForKey(roleKey)
	log.Infof("succeeded to get role cache: %s=%+v", roleKey, roleCache)
	return roleCache, nil
}

// GetJobCache: 获取任务的缓存信息
func (c *MemoryCache) GetJobCache(jobKey string) (JobCache, error) {
	jobCache, err := c.getSubCache(jobKey)
	if err != nil {
		return nil, err
	}
	log := util.LoggerForKey(jobKey)
	log.Infof("succeeded to get job cache: <%s> %+v", jobKey, jobCache)
	return jobCache, nil
}

// DeleteJobCache: 删除任务的缓存信息
func (c *MemoryCache) DeleteJobCache(jobKey string) error {
	log := util.LoggerForKey(jobKey)
	if _, ok := c.cache[jobKey]; ok {
		delete(c.cache, jobKey)
		log.Infof("succeeded to delete job cache: <%s>", jobKey)
	} else {
		log.Infof("job cache has already been deleted: <%s>", jobKey)
	}
	return nil
}
