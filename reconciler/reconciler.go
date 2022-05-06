package reconciler

import (
	"strings"

	"github.com/go-logr/logr"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aresv1 "ares-operator/api/v1"
	"ares-operator/cache"
	config "ares-operator/conf"
	"ares-operator/reconciler/node"
)

var _ aresv1.FrameworkController = &Reconciler{}

type Reconciler struct {
	common.JobController
	client.Client
	aresv1.FrameworkController
	Log              logr.Logger
	Scheme           *runtime.Scheme
	Cache            cache.Interface       // RolePhase缓存
	Config           *config.Configuration // Operator相关配置
	NodeManager      *node.Manager         // 节点管理
	*PodGroupManager                       // PodGroup管理
}

func (r *Reconciler) Close() {
	if r.Cache != nil {
		r.Cache.Close()
	}
}

// Copy: left Controller behind
func (r *Reconciler) Copy() *Reconciler {
	return &Reconciler{
		JobController: common.JobController{
			Controller:                  nil,
			Config:                      r.JobController.Config,
			PodControl:                  r.JobController.PodControl,
			ServiceControl:              r.JobController.ServiceControl,
			KubeClientSet:               r.JobController.KubeClientSet,
			VolcanoClientSet:            r.JobController.VolcanoClientSet,
			PodLister:                   r.JobController.PodLister,
			ServiceLister:               r.JobController.ServiceLister,
			PriorityClassLister:         r.JobController.PriorityClassLister,
			PodInformerSynced:           r.JobController.PodInformerSynced,
			ServiceInformerSynced:       r.JobController.ServiceInformerSynced,
			PriorityClassInformerSynced: r.JobController.PriorityClassInformerSynced,
			Expectations:                r.JobController.Expectations,
			WorkQueue:                   r.JobController.WorkQueue,
			Recorder:                    r.JobController.Recorder,
		},
		Client:          r.Client,
		Log:             r.Log,
		Scheme:          r.Scheme,
		Cache:           r.Cache,
		Config:          r.Config,
		NodeManager:     r.NodeManager,
		PodGroupManager: r.PodGroupManager,
	}
}

func (r *Reconciler) SetController(c aresv1.Controller) {
	r.Controller = c
	r.FrameworkController = c
}

func (r *Reconciler) ResolveOwner(ns string, ref *metav1.OwnerReference) *aresv1.AresJob {
	if ref.Kind != r.GetAPIGroupVersionKind().Kind {
		return nil
	}
	job, err := r.GetJobFromInformerCache(ns, ref.Name)
	if err != nil {
		return nil
	}
	if job.GetUID() != ref.UID {
		return nil
	}
	return job.(*aresv1.AresJob)
}

func (r *Reconciler) GenLabels(jobName string) map[string]string {
	labelGroupName := aresv1.GroupNameLabel
	labelJobName := aresv1.JobNameLabel
	groupName := r.Controller.GetGroupNameLabelValue()
	return map[string]string{
		labelGroupName: groupName,
		labelJobName:   strings.Replace(jobName, "/", "-", -1),
	}
}

// CleanForJob: 清理有关Job残留的资源
func (r *Reconciler) CleanForJob(job *aresv1.AresJob) {
	// delete expectations; 避免数据残留
	r.DeleteExpectations(job)
}

func (r *Reconciler) ControllerName() string {
	return ""
}

func (r *Reconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return aresv1.GroupVersionKind
}

func (r *Reconciler) GetAPIGroupVersion() schema.GroupVersion {
	return aresv1.GroupVersion
}

func (r *Reconciler) GetGroupNameLabelValue() string {
	return aresv1.GroupVersion.Group
}

// SetClusterSpec sets the cluster spec for the pod
func (r *Reconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}

// GetDefaultContainerName 默认容器名
func (r *Reconciler) GetDefaultContainerName() string {
	return ""
}

// GetDefaultContainerPortName 获取默认的容器端口
func (r *Reconciler) GetDefaultContainerPortName() string {
	return ""
}

// IsMasterRole 如果当前角色为master，在创建pod时增加master标签
func (r *Reconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, rtype commonv1.ReplicaType, index int) bool {
	return false
}
