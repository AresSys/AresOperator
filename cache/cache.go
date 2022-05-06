package cache

import (
	"fmt"
	"net/url"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

// CacheOptions: 缓存选项
type CacheOptions struct {
	URI string
}

type Cache struct {
	backend Interface
}

func (c *Cache) SetRoleCache(jobKey string, roleName commonv1.ReplicaType, data *RoleCache) error {
	return c.backend.SetRoleCache(jobKey, roleName, data)
}
func (c *Cache) GetRoleCache(jobKey string, roleName commonv1.ReplicaType) (*RoleCache, error) {
	return c.backend.GetRoleCache(jobKey, roleName)
}
func (c *Cache) GetJobCache(jobKey string) (JobCache, error) { return c.backend.GetJobCache(jobKey) }
func (c *Cache) DeleteJobCache(jobKey string) error          { return c.backend.DeleteJobCache(jobKey) }
func (c *Cache) Close()                                      { c.backend.Close() }

func New(opts *CacheOptions) (Interface, error) {
	u, err := url.Parse(opts.URI)
	if err != nil {
		return nil, err
	}
	var backend Interface
	switch u.Scheme {
	case "memory":
		backend = NewMemoryCache()
	default:
		return nil, fmt.Errorf("unknown cache backend: %v", u.Scheme)
	}
	c := &Cache{backend: backend}
	return c, nil
}
