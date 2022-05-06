package frameworks

import (
	"sync"

	aresv1 "ares-operator/api/v1"
	"ares-operator/frameworks/common"
)

type frameworkFactory struct {
	sync.Mutex
	Builders map[aresv1.FrameworkType]common.FrameworkBuilder
}

func (f *frameworkFactory) Register(name aresv1.FrameworkType, builder common.FrameworkBuilder) {
	f.Lock()
	defer f.Unlock()
	f.Builders[name] = builder
}

var Factory = frameworkFactory{
	Builders: map[aresv1.FrameworkType]common.FrameworkBuilder{},
}

func RegisterBuilder(name aresv1.FrameworkType, builder common.FrameworkBuilder) {
	Factory.Register(name, builder)
}

func init() {
}
