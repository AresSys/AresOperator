package bagua

import (
	"fmt"
	"strconv"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	aresv1 "ares-operator/api/v1"
	fcommon "ares-operator/frameworks/common"
	"ares-operator/reconciler"
	"ares-operator/utils"
)

var _ fcommon.Framework = &Bagua{}

var BaguaBuilder fcommon.FrameworkBuilder = func(r *reconciler.Reconciler, mgr ctrl.Manager) fcommon.Framework {
	return &Bagua{Reconciler: r}
}

type Bagua struct {
	*reconciler.Reconciler
}

func (b *Bagua) GetName() aresv1.FrameworkType {
	return aresv1.FrameworkBagua
}

func (b *Bagua) GetDefaultContainerName() string {
	return b.GetName().String()
}

func (b *Bagua) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	aresJob := job.(*aresv1.AresJob)

	if err := b.addEnv(aresJob, podTemplate, rtype, index); err != nil {
		return err
	}

	addContainerPort(aresJob, podTemplate, rtype, index)

	return nil
}

func (b *Bagua) Validate(job *aresv1.AresJob) error {
	ports := job.Spec.AvailablePorts
	if !enableElastic(job) {
		if err := job.Spec.CheckRolesExist([]commonv1.ReplicaType{aresv1.RoleMaster}); err != nil {
			return err
		}

		if job.Spec.GetRoleReplicas(aresv1.RoleMaster) != 1 {
			return fmt.Errorf("bagua master role replicas must be 1")
		}

		if len(ports) < 2 {
			return fmt.Errorf("insufficient ports(%v), 2 is needed at least under static mode", ports)
		}
	} else {
		if err := job.Spec.CheckRolesExist([]commonv1.ReplicaType{aresv1.RoleWorker, aresv1.RoleEtcd}); err != nil {
			return err
		}
		if job.Spec.GetRoleReplicas(aresv1.RoleEtcd) != 1 {
			return fmt.Errorf("bagua etcd role replicas must be 1")
		}

		minPortsNum := 2 + job.Spec.GetRoleReplicas(aresv1.RoleWorker)
		if len(ports) < int(minPortsNum) {
			return fmt.Errorf("insufficient ports(%v), "+
				"%v is needed at least under static mode", ports, minPortsNum)
		}
	}

	return nil
}

func (b *Bagua) SetDefault(job *aresv1.AresJob) {
	// bagua spec
	if job.Spec.Framework.Bagua == nil {
		job.Spec.Framework.Bagua = &aresv1.BaguaSpec{
			EnableElastic: false,
		}
	}

	if !enableElastic(job) {
		// jobPhasePolicy
		if job.Spec.JobPhasePolicy == nil {
			job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
				//master成功则任务成功
				Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleMaster}},
				//master失败则任务失败
				Failed: [][]commonv1.ReplicaType{{aresv1.RoleMaster}},
			}
			if _, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
				job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
					//worker和master都成功则任务成功
					Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleWorker, aresv1.RoleMaster}},
					//worker和master任一角色失败则任务失败
					Failed: [][]commonv1.ReplicaType{{aresv1.RoleWorker}, {aresv1.RoleMaster}},
				}
			}
			// rolePhasePolicy
			any := aresv1.RolePhaseConditionAnyPod
			all := aresv1.RolePhaseConditionAllPods
			if spec := job.Spec.RoleSpecs[aresv1.RoleMaster]; spec.RolePhasePolicy.Succeeded == nil {
				spec.RolePhasePolicy.Succeeded = &all
			}
			if spec := job.Spec.RoleSpecs[aresv1.RoleMaster]; spec.RolePhasePolicy.Failed == nil {
				spec.RolePhasePolicy.Failed = &any
			}
			if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok {
				if spec.RolePhasePolicy.Succeeded == nil {
					spec.RolePhasePolicy.Succeeded = &all
				}
				if spec.RolePhasePolicy.Failed == nil {
					spec.RolePhasePolicy.Failed = &any
				}
			}
			// dependence
			if spec, ok := job.Spec.RoleSpecs[aresv1.RoleWorker]; ok && spec.Dependence == nil {
				spec.Dependence = &aresv1.DependenceSpec{
					RoleNames: []commonv1.ReplicaType{aresv1.RoleMaster},
					RolePhase: aresv1.RolePhaseRunning,
				}
			}
		}
	} else {
		// jobPhasePolicy
		if job.Spec.JobPhasePolicy == nil {
			job.Spec.JobPhasePolicy = &aresv1.JobPhasePolicy{
				//worker成功则任务成功
				Succeeded: [][]commonv1.ReplicaType{{aresv1.RoleWorker}},
				//worker和etcd任一角色失败则任务失败
				Failed: [][]commonv1.ReplicaType{{aresv1.RoleWorker}, {aresv1.RoleEtcd}},
			}
		}
		// rolePhasePolicy
		any := aresv1.RolePhaseConditionAnyPod
		all := aresv1.RolePhaseConditionAllPods
		if spec := job.Spec.RoleSpecs[aresv1.RoleWorker]; spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
		if spec := job.Spec.RoleSpecs[aresv1.RoleWorker]; spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
		}
		if spec := job.Spec.RoleSpecs[aresv1.RoleEtcd]; spec.RolePhasePolicy.Succeeded == nil {
			spec.RolePhasePolicy.Succeeded = &all
		}
		if spec := job.Spec.RoleSpecs[aresv1.RoleEtcd]; spec.RolePhasePolicy.Failed == nil {
			spec.RolePhasePolicy.Failed = &any
		}
		// dependence
		if spec := job.Spec.RoleSpecs[aresv1.RoleWorker]; spec.Dependence == nil {
			spec.Dependence = &aresv1.DependenceSpec{
				RoleNames: []commonv1.ReplicaType{aresv1.RoleEtcd},
				RolePhase: aresv1.RolePhaseRunning,
			}
		}
	}
}

func (b *Bagua) addEnv(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	var env []corev1.EnvVar

	if enableElastic(job) {
		env = b.getEnvUnderElasticMode(job, podTemplate, rtype, index)
	} else {
		env = b.getEnvUnderStaticMode(job, podTemplate, rtype, index)
	}

	//注入环境变量
	for i, c := range podTemplate.Spec.Containers {
		c.Env = append(c.Env, env...)
		podTemplate.Spec.Containers[i] = c
	}

	return nil
}

func (b *Bagua) getEnvUnderStaticMode(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) []corev1.EnvVar {
	if rtype != string(aresv1.RoleWorker) && rtype != string(aresv1.RoleMaster) {
		return nil
	}

	ports := job.Spec.AvailablePorts
	portStrs := utils.Int32SliceToStrSlice(ports)

	rank := "0"
	if rtype == string(aresv1.RoleWorker) {
		idx, _ := strconv.Atoi(index)
		rank = strconv.FormatInt(int64(idx+1), 10)
	}

	env := []corev1.EnvVar{
		{
			Name:  "BAGUA_SSH_PORT",
			Value: portStrs[0],
		},
		{
			Name: "SERVICE_DOMAIN_NAME",
			Value: strings.Join([]string{
				reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleWorker),
				reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleLauncher),
			}, " "),
		},
		{
			Name: "LOCAL_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
		{
			Name:  "WORLD_SIZE",
			Value: fmt.Sprintf("%v", job.Spec.GetRoleReplicas(aresv1.RoleWorker)+1),
		},
		{
			Name:  "MASTER_ADDR",
			Value: fmt.Sprintf("%v", reconciler.GetPodDomainName(job.Name, job.Namespace, aresv1.RoleMaster, 0)),
		},
		{
			Name:  "MASTER_PORT",
			Value: fmt.Sprintf("%v", portStrs[1]),
		},
		{
			Name:  "RANK",
			Value: fmt.Sprintf("%v", rank),
		},
		{
			Name:  "BAGUA_NODE_DOMAIN_NAMES",
			Value: strings.Join(reconciler.GetRolesDomainNames(job, aresv1.RoleMaster, aresv1.RoleWorker), ","),
		},
		{
			Name:  "BAGUA_AVAILABLE_PORTS",
			Value: strings.Join(portStrs[2:], ","),
		},
	}

	return env
}

func (b *Bagua) getEnvUnderElasticMode(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) []corev1.EnvVar {
	if rtype != string(aresv1.RoleWorker) && rtype != string(aresv1.RoleEtcd) {
		return nil
	}

	ports := job.Spec.AvailablePorts
	portStrs := utils.Int32SliceToStrSlice(ports)

	env := []corev1.EnvVar{
		{
			Name:  "ETCD_CLIENT_PORT",
			Value: portStrs[0],
		},
		{
			Name:  "ETCD_SERVER_PORT",
			Value: portStrs[1],
		},
	}

	if rtype == string(aresv1.RoleEtcd) {
		return env
	}

	workerReplicas := job.Spec.GetRoleReplicas(aresv1.RoleWorker)
	minsize := workerReplicas
	if job.Spec.Framework.Bagua.MinReplicas != nil {
		minsize = *job.Spec.Framework.Bagua.MinReplicas
	}
	maxsize := workerReplicas
	if job.Spec.Framework.Bagua.MaxReplicas != nil {
		maxsize = *job.Spec.Framework.Bagua.MaxReplicas
	}

	env = append(env, []corev1.EnvVar{
		{
			Name:  "SERVICE_DOMAIN_NAME",
			Value: reconciler.GetServiceDomainName(job.Name, job.Namespace, aresv1.RoleWorker),
		},
		{
			Name:  "MIN_SIZE",
			Value: strconv.FormatInt(int64(minsize), 10),
		},
		{
			Name:  "MAX_SIZE",
			Value: strconv.FormatInt(int64(maxsize), 10),
		},
		{
			Name:  "JOB_ID",
			Value: job.Name,
		},
		{
			Name:  "ETCD_HOST",
			Value: reconciler.GetPodDomainName(job.Name, job.Namespace, aresv1.RoleEtcd, 0),
		},
	}...)

	//把剩余端口按照index平均分给每个pod
	privatePorts := utils.Int32SliceToStrSlice(getPrivatePorts(ports[2:], workerReplicas, index))

	if len(privatePorts) > 0 {
		env = append(env, []corev1.EnvVar{
			{
				Name:  "BAGUA_SSH_PORT",
				Value: privatePorts[0],
			},
			{
				Name:  "BAGUA_AVAILABLE_PORTS",
				Value: strings.Join(privatePorts[1:], ","),
			},
		}...)
	}

	return env
}

//把除了etcd之外的端口按照index平均分给每个pod
func getPrivatePorts(portsExceptEtcd []int32, replicas int32, index string) (privatePorts []int32) {
	privatePortNumEach := len(portsExceptEtcd) / int(replicas)
	idx, _ := strconv.Atoi(index)
	if idx < int(replicas) && privatePortNumEach > 0 {
		privatePorts = portsExceptEtcd[privatePortNumEach*idx : privatePortNumEach*(idx+1)]
	}

	return privatePorts
}

func addContainerPort(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) {
	containerPorts := []corev1.ContainerPort{}

	allPorts := job.Spec.AvailablePorts
	addPorts := []int32{}

	if !enableElastic(job) {
		addPorts = allPorts
	} else {
		if rtype == string(aresv1.RoleEtcd) {
			addPorts = allPorts[:2]
		} else {
			addPorts = getPrivatePorts(allPorts[2:], job.Spec.GetRoleReplicas(aresv1.RoleWorker), index)
		}
	}

	for _, p := range addPorts {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: int32(p),
		})
	}

	podTemplate.Spec.Containers[0].Ports = append(podTemplate.Spec.Containers[0].Ports, containerPorts...)
}

func enableElastic(job *aresv1.AresJob) bool {
	return job.Spec.Framework.Bagua.EnableElastic
}
