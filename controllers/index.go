package controllers

import (
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler"
)

func IndexObjectOwner(rawObj client.Object) []string {
	obj := rawObj.(metav1.Object)
	owner := metav1.GetControllerOf(obj)
	if !aresv1.IsAresJob(owner) {
		return nil
	}
	// We do not need to add namespace, cause controller-runtime will add it for us
	// Refer to: sigs.k8s.io/controller-runtime/pkg/cache/informer_cache.go
	return []string{owner.Name}
}

func IndexReplicaOwner(rawObj client.Object) []string {
	obj := rawObj.(metav1.Object)
	owner := metav1.GetControllerOf(obj)
	if !aresv1.IsAresJob(owner) {
		return nil
	}
	if rtype := reconciler.GetReplicaType(obj.GetLabels()); len(rtype) > 0 {
		return []string{aresv1.MetaReplicaKeyFormat(owner.Name, commonv1.ReplicaType(rtype))}
	}
	return nil
}
