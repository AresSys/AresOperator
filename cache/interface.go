package cache

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

type Interface interface {
	SetRoleCache(jobKey string, roleName commonv1.ReplicaType, data *RoleCache) error
	GetRoleCache(jobKey string, roleName commonv1.ReplicaType) (*RoleCache, error)
	GetJobCache(jobKey string) (JobCache, error)
	DeleteJobCache(jobKey string) error
	Close()
}
