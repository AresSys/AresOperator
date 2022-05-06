package control

import (
	"context"
	"fmt"

	"github.com/kubeflow/common/pkg/controller.v1/control"
	commonutil "github.com/kubeflow/common/pkg/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// RealPodControl is the default implementation of PodControlInterface.
type RealPodControl struct {
	*control.RealPodControl
}

func (r RealPodControl) CreatePods(namespace string, template *v1.PodTemplateSpec, object runtime.Object) error {
	return r.createPods("", namespace, template, object, nil)
}

func (r RealPodControl) CreatePodsWithControllerRef(namespace string, template *v1.PodTemplateSpec, controllerObject runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := control.ValidateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods("", namespace, template, controllerObject, controllerRef)
}

func (r RealPodControl) CreatePodsOnNode(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	if err := control.ValidateControllerRef(controllerRef); err != nil {
		return err
	}
	return r.createPods(nodeName, namespace, template, object, controllerRef)
}

func (r RealPodControl) createPods(nodeName, namespace string, template *v1.PodTemplateSpec, object runtime.Object, controllerRef *metav1.OwnerReference) error {
	pod, err := control.GetPodFromTemplate(template, object, controllerRef)
	if err != nil {
		return err
	}
	// NOTE: 添加GenerateName
	pod.GenerateName = template.GenerateName
	if len(nodeName) != 0 {
		pod.Spec.NodeName = nodeName
	}
	if labels.Set(pod.Labels).AsSelectorPreValidated().Empty() {
		return fmt.Errorf("unable to create pods, no labels")
	}
	logger := commonutil.LoggerForPod(pod, object.GetObjectKind().GroupVersionKind().Kind)
	if newPod, err := r.KubeClient.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		r.Recorder.Eventf(object, v1.EventTypeWarning, control.FailedCreatePodReason, "Error creating: %v", err)
		return err
	} else {
		accessor, err := meta.Accessor(object)
		if err != nil {
			logger.Errorf("parentObject does not have ObjectMeta, %v", err)
			return nil
		}
		logger.Infof("Controller %v created pod %v", accessor.GetName(), newPod.Name)
		r.Recorder.Eventf(object, v1.EventTypeNormal, control.SuccessfulCreatePodReason, "Created pod: %v", newPod.Name)
	}
	return nil
}
