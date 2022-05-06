package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"

	aresv1 "ares-operator/api/v1"
)

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()
	jobKey := "default.unittest"
	roleCache := NewRoleCache(aresv1.RolePhaseRunning)
	// SetRoleCache
	err := c.SetRoleCache(jobKey, aresv1.RoleWorker, roleCache)
	assert.Nil(t, err)

	// GetRoleCache
	roleResult, err := c.GetRoleCache(jobKey, aresv1.RoleWorker)
	assert.Nil(t, err)
	t.Logf("GetRoleCache: %+v", roleResult)
	assert.Equal(t, roleCache, roleResult)

	// GetJobCache
	jobResult, err := c.GetJobCache(jobKey)
	assert.Nil(t, err)
	t.Logf("GetJobCache: %+v", jobResult)
	assert.Equal(t, JobCache{aresv1.RoleWorker: roleCache}, jobResult)

	// DeleteJobCache
	err = c.DeleteJobCache(jobKey)
	assert.Nil(t, err)
	assert.Nil(t, c.cache[jobKey])

	// Close
	c.Close()
	assert.Nil(t, c.cache)
}
