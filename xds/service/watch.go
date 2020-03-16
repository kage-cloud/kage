package service

import (
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/util/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type WatchService interface {
	DeploymentPods(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error
	DeploymentServices(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error
}

type watchService struct {
	KubeClient kube.Client `inject:"KubeClient"`
}

func (w *watchService) DeploymentServices(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error {
	ns := kubeutil.DeploymentPodNamespace(deploy)

	svcs, wi := w.KubeClient.InformAndListServices(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok && v.Namespace == ns {
			selector := labels.SelectorFromSet(v.Spec.Selector)
			return selector.Matches(labels.Set(deploy.Spec.Template.Labels))
		}
		return false
	})
	svcList := &corev1.ServiceList{
		Items: svcs,
	}

	for _, eh := range eventHandler {
		if err := eh.OnListEvent(svcList); err != nil {
			return err
		}
	}

	go func() {
		for e := range wi {
			for _, handler := range eventHandler {
				handler.OnWatchEvent(e)
			}
		}
	}()
	return nil
}

func (w *watchService) DeploymentPods(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	ns := deploy.Spec.Template.Namespace
	if ns == "" {
		ns = deploy.Namespace
	}

	pods, wi := w.KubeClient.InformAndListPod(func(object metav1.Object) bool {
		return object.GetNamespace() == ns && selector.Matches(labels.Set(object.GetLabels()))
	})
	podList := &corev1.PodList{
		Items: pods,
	}

	for _, eh := range eventHandler {
		if err := eh.OnListEvent(podList); err != nil {
			return err
		}
	}

	go func() {
		for e := range wi {
			for _, handler := range eventHandler {
				handler.OnWatchEvent(e)
			}
		}
	}()
	return nil
}
