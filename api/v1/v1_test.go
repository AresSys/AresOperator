package v1

import (
	"testing"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetRolesIntersection(t *testing.T) {
	roleSpecs := map[commonv1.ReplicaType]*RoleSpec{
		RoleMaster: nil,
		RoleWorker: nil,
		RoleEtcd:   nil,
	}

	intersection := GetRolesIntersection(roleSpecs, []commonv1.ReplicaType{RoleWorker, RoleChief, RoleEtcd})
	assert.Equal(t, []commonv1.ReplicaType{RoleWorker, RoleEtcd}, intersection)
}
