package frameworks

import (
	"sync"

	aresv1 "ares-operator/api/v1"
	"ares-operator/frameworks/bagua"
	"ares-operator/frameworks/common"
	"ares-operator/frameworks/custom"
	"ares-operator/frameworks/dask"
	"ares-operator/frameworks/demo"
	"ares-operator/frameworks/mpi"
	"ares-operator/frameworks/pytorch"
	"ares-operator/frameworks/standalone"
	"ares-operator/frameworks/tensorflow"
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
	RegisterBuilder(aresv1.FrameworkBagua, bagua.BaguaBuilder)
	RegisterBuilder(aresv1.FrameworkCustom, custom.CustomBuilder)
	RegisterBuilder(aresv1.FrameworkDask, dask.DaskBuilder)
	RegisterBuilder(aresv1.FrameworkDemo, demo.DemoBuilder)
	RegisterBuilder(aresv1.FrameworkMPI, mpi.MPIBuilder)
	RegisterBuilder(aresv1.FrameworkPytorch, pytorch.PytorchBuilder)
	RegisterBuilder(aresv1.FrameworkStandalone, standalone.StandaloneBuilder)
	RegisterBuilder(aresv1.FrameworkTensorflow, tensorflow.TensorflowBuilder)
}
