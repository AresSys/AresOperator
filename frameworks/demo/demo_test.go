package demo

import (
	"testing"

	"ares-operator/reconciler"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	aresv1 "ares-operator/api/v1"
)

func TestDemoFramework_Reconcile(t *testing.T) {
	f := &DemoFramework{
		Reconciler: &reconciler.Reconciler{
			Log: zap.New(),
		},
	}
	f.SetController(f)

	err := f.ReconcileJobs(nil, nil, commonv1.JobStatus{}, nil)
	assert.Nil(t, err)

	err = f.Controller.ReconcileServices(nil, nil, aresv1.RoleWorker, nil)
	assert.Nil(t, err)
}
