package reconciler

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	commonutil "github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aresv1 "ares-operator/api/v1"
)

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *Reconciler) GetServicesForJob(job interface{}) ([]*corev1.Service, error) {
	obj, err := meta.Accessor(job)
	if err != nil {
		return nil, fmt.Errorf("%+v is not a type of AresJob", job)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	if err := r.List(context.Background(), serviceList, client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{aresv1.JobOwnerKey: obj.GetName()}); err != nil {
		return nil, err
	}

	//TODO support adopting/orphaning
	ret := convertServiceList(serviceList.Items)

	return ret, nil
}

// convertServiceList convert service list to service point list
func convertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

// FilterServicesForReplicaType returns service belong to a replicaType.
func (r *Reconciler) FilterServicesForReplicaType(services []*corev1.Service, replicaType string) ([]*corev1.Service, error) {
	var result []*corev1.Service

	replicaSelector := labels.SelectorFromSet(labels.Set{aresv1.ReplicaTypeLabel: replicaType})
	deprecatedSelector := labels.SelectorFromSet(labels.Set{commonv1.ReplicaTypeLabel: replicaType})

	for _, service := range services {
		if !deprecatedSelector.Matches(labels.Set(service.Labels)) && !replicaSelector.Matches(labels.Set(service.Labels)) {
			continue
		}
		result = append(result, service)
	}
	return result, nil
}

// 对JobController结构体方法 ReconcileServices（https://github.com/kubeflow/common/blob/master/pkg/controller.v1/common/service.go#L206）进行了重新定义
// 每个角色对应一个service
func (r *Reconciler) ReconcileServices(
	job metav1.Object,
	services []*corev1.Service,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	// Get all services for the type rt.
	services, err := r.FilterServicesForReplicaType(services, rt)
	if err != nil {
		return err
	}

	log := commonutil.LoggerForReplica(job, rt)
	// 一个角色只有一个service
	if len(services) > 1 {
		log.Warningf("We have too many services(%v) for %s", len(services), rt)
	} else if len(services) == 0 {
		commonutil.LoggerForReplica(job, rt).Infof("need to create new service: %s", rt)
		err = r.CreateNewSharedService(job, rtype, spec)
		if err != nil {
			log.Errorf("failed to create service: %v", err)
			return err
		}
	}

	return nil
}

// GenerateServiceName generates service name for given role
func (r *Reconciler) GenerateServiceName(jobName, roleName string) string {
	return GetGeneralSharedName(jobName, roleName)
}

// CreateNewSharedService creates a new service for the given type.
// 参考CreateNewService方法 https://github.com/kubeflow/common/blob/master/pkg/controller.v1/common/service.go#L277
func (r *Reconciler) CreateNewSharedService(job metav1.Object, rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {
	jobKey, err := common.KeyFunc(job)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for job object %#v: %v", job, err))
		return err
	}

	// Convert ReplicaType to lower string.
	rt := strings.ToLower(string(rtype))

	// Append ReplicaTypeLabel labels.
	labels := r.GenLabels(job.GetName())
	labels[aresv1.ReplicaTypeLabel] = rt

	ports, err := r.GetPortsFromJob(spec)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports:     []corev1.ServicePort{},
		},
	}

	// Add service ports to headless service
	for name, port := range ports {
		svcPort := corev1.ServicePort{Name: name, Port: port}
		service.Spec.Ports = append(service.Spec.Ports, svcPort)
	}

	if len(service.Spec.Ports) == 0 {
		service.Spec.Ports = []corev1.ServicePort{
			{
				Port:       80,
				TargetPort: intstr.FromInt(80),
			},
		}
	}

	service.Name = r.FrameworkController.GenerateServiceName(job.GetName(), rt)
	service.Labels = labels
	// Create OwnerReference.
	controllerRef := r.GenOwnerReference(job)

	expectationServicesKey := expectation.GenExpectationServicesKey(jobKey, rt)
	r.Expectations.RaiseExpectations(expectationServicesKey, 1, 0)
	err = r.ServiceControl.CreateServicesWithControllerRef(job.GetNamespace(), service, job.(runtime.Object), controllerRef)
	if err != nil && errors.IsTimeout(err) {
		// Service is created but its initialization has timed out.
		// If the initialization is successful eventually, the
		// controller will observe the creation via the informer.
		// If the initialization fails, or if the service keeps
		// uninitialized for a long time, the informer will not
		// receive any update, and the controller will create a new
		// service when the expectation expires.

		//succeededServiceCreationCount和failedServiceCreationCount于https://github.com/kubeflow/common/blob/master/pkg/controller.v1/common/service.go#L36定义
		//我们暂未使用prometheus，因此不进行统计
		// succeededServiceCreationCount.Inc()
		return nil
	} else if err != nil {
		// failedServiceCreationCount.Inc()
		r.Expectations.CreationObserved(expectationServicesKey)
		return err
	}
	// succeededServiceCreationCount.Inc()
	return nil
}

// GetPortsFromJob 获取指定容器对应的端口，如果容器名为空，返回第一个容器的端口
func (r *Reconciler) GetPortsFromJob(spec *commonv1.ReplicaSpec) (map[string]int32, error) {
	defaultContainerName := r.Controller.GetDefaultContainerName()
	if defaultContainerName != "" {
		return r.JobController.GetPortsFromJob(spec)
	}

	ports := make(map[string]int32)
	containerPorts := spec.Template.Spec.Containers[0].Ports
	if len(containerPorts) == 0 {
		return nil, nil
	}
	for _, port := range containerPorts {
		ports[port.Name] = port.ContainerPort
	}
	return ports, nil
}
