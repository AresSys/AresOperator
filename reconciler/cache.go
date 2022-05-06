package reconciler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) getObserverHostIPs() ([]string, error) {
	c := r.Config.Cache.Observer
	list := &corev1.PodList{}
	if err := r.List(context.Background(), list, client.InNamespace(c.Namespace),
		client.MatchingLabels(c.Labels)); err != nil {
		return nil, err
	}
	pods := convertPodList(list.Items)
	ips := sets.NewString()
	for _, pod := range pods {
		ip := pod.Status.HostIP
		if len(ip) > 0 {
			ips.Insert(ip)
		}
	}
	return ips.List(), nil
}

func (r *Reconciler) getCacheURI() string {
	c := r.Config.Cache
	if !c.NodePortEnabled() {
		return c.GetURI()
	}
	ips, err := r.getObserverHostIPs()
	if err != nil {
		return c.Observer.NodePortURI()
	}
	return c.Observer.NodePortURI(ips...)
}
