package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

	aresv1 "ares-operator/api/v1"
	"ares-operator/reconciler"
	"ares-operator/reconciler/node"
)

/***************************
 * Object: Pod/Service/...
 ***************************/
func (r *AresJobReconciler) AddObject(e event.CreateEvent) bool {
	ref := metav1.GetControllerOf(e.Object)
	if ref == nil {
		return true
	}
	job := r.Reconciler.ResolveOwner(e.Object.GetNamespace(), ref)
	if job == nil {
		return true
	}
	f := r.getFramework(job.Spec.FrameworkType)
	if f == nil {
		return false
	}
	f.AddObject(job, e.Object)
	return true
}

// UpdateObject: 过滤无价值事件
func (r *AresJobReconciler) UpdateObject(e event.UpdateEvent) bool {
	_, obj := e.ObjectOld, e.ObjectNew
	if pod, ok := obj.(*corev1.Pod); ok {
		if conditions := pod.Status.Conditions; len(conditions) == 1 {
			cond := conditions[0]
			if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse {
				return false
			}
		}
	} else if pg, ok := obj.(*scheduling.PodGroup); ok {
		if conditions := pg.Status.Conditions; len(conditions) == 1 {
			cond := conditions[0]
			if cond.Type == scheduling.PodGroupUnschedulableType && cond.Status == corev1.ConditionTrue {
				return false
			}
		}
	}
	return true
}

func (r *AresJobReconciler) DeleteObject(e event.DeleteEvent) bool {
	ref := metav1.GetControllerOf(e.Object)
	if ref == nil {
		return true
	}

	job := r.Reconciler.ResolveOwner(e.Object.GetNamespace(), ref)
	if job == nil {
		return true
	}

	f := r.getFramework(job.Spec.FrameworkType)
	if f == nil {
		return false
	}
	f.DeleteObject(job, e.Object)
	return true
}

/**********
 * Node
 **********/
// NodeEventHandler: Node事件处理
type NodeEventHandler struct {
	manager *node.Manager
	r       *reconciler.Reconciler
}

// Create implements EventHandler
func (h *NodeEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	node := evt.Object.(*corev1.Node)
	h.manager.RemoveNode(node.Name)
}

// Update implements EventHandler
func (h *NodeEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	old := evt.ObjectOld.(*corev1.Node)
	node := evt.ObjectNew.(*corev1.Node)
	nodeDown := h.manager.HandleUpdate(old, node)
	if !nodeDown {
		return
	}
	h.handlePods(node, q)
}

// Delete implements EventHandler
func (h *NodeEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	node := evt.Object.(*corev1.Node)
	h.manager.RemoveNode(node.Name)
}

func (h *NodeEventHandler) handlePods(node *corev1.Node, q workqueue.RateLimitingInterface) {
	pods, _ := h.r.GetPodsForNode(node.Name)
	for _, pod := range pods {
		owner := metav1.GetControllerOf(pod)
		if !aresv1.IsAresJob(owner) {
			continue
		}
		req := reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      owner.Name,
			Namespace: pod.Namespace,
		}}
		q.Add(req)
	}
}

// Generic implements EventHandler
func (h *NodeEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	node := evt.Object.(*corev1.Node)
	nodeDown := h.manager.HandleGeneric(node)
	if !nodeDown {
		return
	}
	h.handlePods(node, q)
}
