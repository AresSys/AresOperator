package dask

import (
	"fmt"
	"strconv"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
)

var _ fcommon.Framework = &Dask{}

var DaskBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &Dask{Reconciler: r}
}

type Dask struct {
	*reconciler.Reconciler
}

func (d *Dask) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkDask
}

func (d *Dask) GetDefaultContainerName() string {
	return d.GetName().String()
}

func (d *Dask) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	aresJob := job.(*aresv1.AresJob)
	if err := d.addEnv(aresJob, podTemplate, rtype, index); err != nil {
		return err
	}
	return nil
}

func (d *Dask) addEnv(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	env := d.getEnvForDaskRole(job, podTemplate, rtype, index)
	//注入环境变量
	for i, c := range podTemplate.Spec.Containers {
		c.Env = append(c.Env, env...)
		podTemplate.Spec.Containers[i] = c
	}
	return nil
}

func (d *Dask) getRoleDefaultContainerPort(job *aresv1.AresJob, rtype commonv1.ReplicaType) string {
	if role, ok := job.Spec.RoleSpecs[rtype]; ok {
		ports := d.getPodDefaultContainerPorts(&role.Template)
		if ports != nil {
			return strconv.FormatInt(int64(ports[0].ContainerPort), 10)
		}
		return ""
	}
	return ""
}

func (d *Dask) getPodDefaultContainerPorts(podTemplate *corev1.PodTemplateSpec) []corev1.ContainerPort {
	for _, container := range podTemplate.Spec.Containers {
		if container.Name == d.GetDefaultContainerName() {
			return container.Ports
		}
	}
	return nil
}

func (d *Dask) getEnvForDaskRole(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) []corev1.EnvVar {
	var envs []corev1.EnvVar
	schedulerDomain := reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleScheduler)
	schedulerPort := d.getRoleDefaultContainerPort(job, aresv1.RoleScheduler)
	if rtype == string(aresv1.RoleScheduler) {
		envs = append(envs, corev1.EnvVar{
			Name: "MY_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		})
		envs = append(envs, corev1.EnvVar{
			Name:  "SCHEDULER",
			Value: fmt.Sprintf("tcp://%s:%s", schedulerDomain, schedulerPort),
		})
		return envs
	}

	masterPort := d.getRoleDefaultContainerPort(job, aresv1.RoleLeadWorker)
	if masterPort == "" {
		masterPort = d.getRoleDefaultContainerPort(job, aresv1.RoleLauncher)
	}
	rank := 0
	worldSize := d.getWorldSize(job)
	masterAddr := "0.0.0.0"
	if rtype == string(aresv1.RoleWorker) {
		idx, _ := strconv.Atoi(index)
		rank = idx + 1
		masterAddr = reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleLeadWorker)
	}
	if rtype == string(aresv1.RoleLauncher) {
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleLeadWorker]; ok {
			masterAddr = reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleLeadWorker)
			rank = worldSize - 1
		}
		envs = append(envs, corev1.EnvVar{
			Name:  "SCHEDULER",
			Value: fmt.Sprintf("tcp://%s:%s", schedulerDomain, schedulerPort),
		})
	}
	envs = append(envs, corev1.EnvVar{
		//命名方式兼容非ares版本
		Name:  "SchedulerAddress",
		Value: schedulerDomain,
	})
	envs = append(envs, corev1.EnvVar{
		//命名方式兼容非ares版本
		Name:  "SchedulerPort",
		Value: schedulerPort,
	})
	envs = append(envs, corev1.EnvVar{
		Name:  "RANK",
		Value: strconv.Itoa(rank),
	})
	envs = append(envs, corev1.EnvVar{
		Name:  "MASTER_ADDR",
		Value: masterAddr,
	})
	envs = append(envs, corev1.EnvVar{
		Name:  "WORLD_SIZE",
		Value: strconv.Itoa(worldSize),
	})
	envs = append(envs, corev1.EnvVar{
		Name:  "MASTER_PORT",
		Value: masterPort,
	})
	return envs
}

func (d *Dask) getWorldSize(job *aresv1.AresJob) int {
	worldSize := 0
	for key, role := range job.Spec.RoleSpecs {
		if key == aresv1.RoleLeadWorker || key == aresv1.RoleLauncher || key == aresv1.RoleWorker {
			worldSize = worldSize + int(*role.Replicas)
		}
	}
	return worldSize
}

func (d *Dask) SetDefault(job *aresv1.AresJob) {
	if job.Spec.JobPhasePolicy == nil {
		failedCondition := [][]commonv1.ReplicaType{{aresv1.RoleLauncher}, {aresv1.RoleScheduler}}
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleLeadWorker]; ok {
			failedCondition = append(failedCondition, []commonv1.ReplicaType{aresv1.RoleLeadWorker})
		}
		if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
			failedCondition = append(failedCondition, []commonv1.ReplicaType{aresv1.RoleWorker})
		}
		job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
			// dask launcher 成功则成功
			Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleLauncher}},
			// 任何角色失败则失败
			Failed: failedCondition,
		}
	}

	// rolePhasePolicy
	any := aresv1.RolePhaseConditionAnyPod
	all := aresv1.RolePhaseConditionAllPods
	schedulerSpec := job.Spec.RoleSpecs[aresv1.RoleScheduler]
	if schedulerSpec.RolePhasePolicy.Succeeded == nil {
		schedulerSpec.RolePhasePolicy.Succeeded = &all
	}
	if schedulerSpec.RolePhasePolicy.Failed == nil {
		schedulerSpec.RolePhasePolicy.Failed = &any
	}
	schedulerSpec.Dependence = nil
	schedulerSpec.RestartPolicy = commonv1.RestartPolicyOnFailure
	schedulerSpec.Template.Spec.HostNetwork = true

	launcherDependenceRole := aresv1.RoleScheduler
	launcherSpec := job.Spec.RoleSpecs[aresv1.RoleLauncher]
	if launcherSpec.RolePhasePolicy.Succeeded == nil {
		launcherSpec.RolePhasePolicy.Succeeded = &all
	}
	if launcherSpec.RolePhasePolicy.Failed == nil {
		launcherSpec.RolePhasePolicy.Failed = &any
	}

	if masterSpec, ok := job.Spec.RoleSpecs[aresv1.RoleLeadWorker]; ok {
		if masterSpec.RolePhasePolicy.Succeeded == nil {
			masterSpec.RolePhasePolicy.Succeeded = &all
		}
		if masterSpec.RolePhasePolicy.Failed == nil {
			masterSpec.RolePhasePolicy.Failed = &any
		}
		if masterSpec.Dependence == nil {
			masterSpec.Dependence = &aresv1.DependenceSpec{
				RoleNames: []commonv1.ReplicaType{aresv1.RoleScheduler},
				RolePhase: aresv1.RolePhaseRunning,
			}
		}
		launcherDependenceRole = aresv1.RoleLeadWorker
	}
	if workerSpec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if workerSpec.RolePhasePolicy.Succeeded == nil {
			workerSpec.RolePhasePolicy.Succeeded = &all
		}
		if workerSpec.RolePhasePolicy.Failed == nil {
			workerSpec.RolePhasePolicy.Failed = &any
		}
		if workerSpec.Dependence == nil {
			workerSpec.Dependence = &aresv1.DependenceSpec{
				RoleNames: []commonv1.ReplicaType{aresv1.RoleLeadWorker},
				RolePhase: aresv1.RolePhaseRunning,
			}
		}
		launcherDependenceRole = aresv1.RoleWorker
	}
	launcherSpec.Dependence = &aresv1.DependenceSpec{
		RoleNames: []commonv1.ReplicaType{launcherDependenceRole},
		RolePhase: aresv1.RolePhaseRunning,
	}

}

func (d *Dask) Validate(job *aresv1.AresJob) error {
	if err := job.Spec.CheckRolesExist([]commonv1.ReplicaType{aresv1.RoleLauncher, aresv1.RoleScheduler}); err != nil {
		return err
	}
	if job.Spec.GetRoleReplicas(aresv1.RoleLauncher) != 1 {
		return fmt.Errorf("dask launcher role replicas must be 1")
	}
	if job.Spec.GetRoleReplicas(aresv1.RoleScheduler) != 1 {
		return fmt.Errorf("dask scheduler role replicas must be 1")
	}
	// check necessary pod.
	if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
		if err := job.Spec.CheckRolesExist([]commonv1.ReplicaType{aresv1.RoleLeadWorker}); err != nil {
			return fmt.Errorf("dask master role must exist on the cases of worker role provided")
		}
	}
	if masterSpec, ok := job.Spec.RoleSpecs[aresv1.RoleLeadWorker]; ok {
		if *masterSpec.Replicas != 1 {
			return fmt.Errorf("dask master role replicas must be 1")
		}
		ports := d.getPodDefaultContainerPorts(&masterSpec.Template)
		if len(ports) != 1 {
			return fmt.Errorf("dask master container port number must be 1")
		}
	}
	schedulerPort := d.getPodDefaultContainerPorts(&job.Spec.RoleSpecs[aresv1.RoleScheduler].Template)
	if len(schedulerPort) != 2 {
		return fmt.Errorf("dask scheduler container port number must be 2")
	}
	launcherPorts := d.getPodDefaultContainerPorts(&job.Spec.RoleSpecs[aresv1.RoleLauncher].Template)
	if len(launcherPorts) == 0 {
		return fmt.Errorf("dask launcher container port number can not be 0")
	}
	return nil
}

// ReconcileServices
func (d *Dask) ReconcileServices(
	job metav1.Object,
	services []*corev1.Service,
	rtype commonv1.ReplicaType,
	spec *commonv1.ReplicaSpec) error {
	log := util.LoggerForJob(job)
	if rtype == aresv1.RoleLauncher || rtype == aresv1.RoleWorker {
		log.Debugf("No need create service for: %s/%s", d.GetName(), rtype)
		return nil
	}
	log.Debugf("create service for : %s/%s", d.GetName(), rtype)
	return d.Reconciler.ReconcileServices(job, services, rtype, spec)

}
