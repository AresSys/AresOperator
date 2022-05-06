package v1

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func GetRolesIntersection(roleSpecs map[commonv1.ReplicaType]*RoleSpec, roles []commonv1.ReplicaType) []commonv1.ReplicaType {
	allRoles := sets.NewString()
	for r := range roleSpecs {
		allRoles.Insert(string(r))
	}

	intersection := []commonv1.ReplicaType{}
	for _, r := range roles {
		if !allRoles.Has(string(r)) {
			continue
		}
		intersection = append(intersection, r)
	}
	return intersection
}
