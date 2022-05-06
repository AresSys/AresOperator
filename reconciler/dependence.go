package reconciler

import (
	"bytes"
	"path"
	"strings"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	corev1 "k8s.io/api/core/v1"

	aresv1 "ares-operator/api/v1"
	"ares-operator/cache"
	"ares-operator/conf"
)

// 临时配置存储
func GetConfigurationVolume() corev1.Volume {
	return corev1.Volume{
		Name: "configurations",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// 根据init config和cmd config生成init container
// kai有附加init container需求，因此抽象本函数，方便框架调用
func GetInitContainer(initConfig *conf.InitializerConfig, cmdConfig conf.CmdConfig, setLocalIPEnv bool) (*corev1.Container, error) {
	if cmdConfig.Roles == "" {
		cmdConfig.Roles = "''"
	}
	buffer := &bytes.Buffer{}
	if err := initConfig.CmdTemplate.Execute(buffer, cmdConfig); err != nil {
		return nil, err
	}
	// 构建initContainer
	initContainer := corev1.Container{
		Name:            "initializer",
		Image:           initConfig.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"/bin/bash",
			"-c",
			strings.ReplaceAll(buffer.String(), "\n", " "),
		},
		Env: []corev1.EnvVar{
			{
				Name: "KS_KUBE_HOST_IP",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
		},
	}
	// 环境变量
	if setLocalIPEnv {
		initContainer.Env = append(initContainer.Env, corev1.EnvVar{
			Name: "LOCAL_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		})
	}
	for name, val := range initConfig.Envs {
		initContainer.Env = append(initContainer.Env, corev1.EnvVar{
			Name:  name,
			Value: val,
		})
	}
	// 挂载volumes
	for _, volume := range initConfig.Volumes {
		initContainer.VolumeMounts = append(initContainer.VolumeMounts, corev1.VolumeMount{
			Name:      volume.Name,
			MountPath: volume.HostPath,
		})
	}
	return &initContainer, nil
}

func (r *Reconciler) AddMPIInitializer(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	spec := job.Spec.RoleSpecs[commonv1.ReplicaType(rtype)]
	// 1. 构建临时存储，挂载到container+initContainer上
	// a. 构建volume
	volume := GetConfigurationVolume()
	// b. 构建volumeMount
	filepaths := *spec.Dependence.MPIHostFile
	mountPath := path.Dir(strings.Split(filepaths, ",")[0])
	mount := corev1.VolumeMount{
		Name:      volume.Name,
		MountPath: mountPath,
	}
	// c.添加临时存储到container
	podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, volume)
	podTemplate.Spec.Containers[0].VolumeMounts = append(podTemplate.Spec.Containers[0].VolumeMounts, mount)

	// 2. 使用initContainer
	cmdConfig := conf.CmdConfig{
		ReplicasPerNode:   job.Spec.Framework.MPI.ReplicasPerNode,
		MPIHostFile:       filepaths,
		MPIImplementation: job.Spec.Framework.MPI.Implementation,
	}
	initContainer, err := r.addInitContainer(job, podTemplate, rtype, cmdConfig)
	if err != nil {
		return err
	}
	initContainer.VolumeMounts = append([]corev1.VolumeMount{mount}, initContainer.VolumeMounts...)
	podTemplate.Spec.InitContainers = append([]corev1.Container{*initContainer}, podTemplate.Spec.InitContainers...)
	return nil
}

func (r *Reconciler) addInitContainer(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype string, cmdConfig conf.CmdConfig) (*corev1.Container, error) {
	roles := []string{}
	dependence := job.Spec.RoleSpecs[commonv1.ReplicaType(rtype)].Dependence
	for _, role := range dependence.RoleNames {
		roles = append(roles, string(role))
	}
	cmdConfig.Name = job.Name
	cmdConfig.Namespace = job.Namespace
	cmdConfig.CacheURI = r.getCacheURI()
	cmdConfig.Key = cache.NamespacedNameToKey(job.Namespace, job.Name)
	cmdConfig.Roles = strings.Join(roles, ",")

	initConfig, err := r.Config.GetCommonConfig()
	if err != nil {
		return nil, err
	}

	initContainer, err := GetInitContainer(initConfig, cmdConfig, true)
	if err != nil {
		return nil, err
	}
	// podTemplate增加Volumes申请
	for _, volume := range initConfig.Volumes {
		podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
			Name: volume.Name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volume.HostPath,
				},
			},
		})
	}

	return initContainer, nil
}

func (r *Reconciler) AddWaiter(job *aresv1.AresJob, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	cmdConfig := conf.CmdConfig{
		MPIHostFile:       "''",
		MPIImplementation: "''",
	}
	initContainer, err := r.addInitContainer(job, podTemplate, rtype, cmdConfig)
	if err != nil {
		return err
	}
	podTemplate.Spec.InitContainers = append([]corev1.Container{*initContainer}, podTemplate.Spec.InitContainers...)
	return nil
}
