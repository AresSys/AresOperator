package utils

import (
	"k8s.io/client-go/tools/cache"
)

// NewThreadSafeStore: 新建一个ThreadSafe存储
func NewThreadSafeStore() cache.ThreadSafeStore {
	return cache.NewThreadSafeStore(cache.Indexers{}, cache.Indices{})
}
