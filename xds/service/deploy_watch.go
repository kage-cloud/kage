package service

import (
	"fmt"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/synchelpers"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/util/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const DeployWatchServiceKey = "DeployWatchService"

type DeployWatchService interface {
	DeploymentPods(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error
	DeploymentServices(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error
}

type deployWatchService struct {
	KubeClient            kube.Client           `inject:"KubeClient"`
	StopperHandlerService StopperHandlerService `inject:"StopperHandlerService"`
}

func (d *deployWatchService) DeploymentServices(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error {
	ns := kubeutil.DeploymentPodNamespace(deploy)

	svcs, wi := d.KubeClient.InformAndListServices(func(object metav1.Object) bool {
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

	stopChan := make(chan error)
	objKey := kubeutil.ObjectKey(deploy)
	stopper := synchelpers.NewStopper(func(err error) {
		d.StopperHandlerService.Remove(objKey)
		stopChan <- err
	})
	go func() {
		for {
			select {
			case e := <-wi:
				for _, handler := range eventHandler {
					handler.OnWatchEvent(e)
				}
			case err := <-stopChan:
				if err != nil {
					fmt.Println("Stopping watch of", deploy.Name, "services in", deploy.Namespace)
				}
				return
			}
		}
	}()
	d.StopperHandlerService.Add(objKey, stopper)
	return nil
}

func (d *deployWatchService) DeploymentPods(deploy *appsv1.Deployment, eventHandler ...model.InformEventHandler) error {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	ns := deploy.Spec.Template.Namespace
	if ns == "" {
		ns = deploy.Namespace
	}

	pods, wi := d.KubeClient.InformAndListPod(func(object metav1.Object) bool {
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

	objKey := kubeutil.ObjectKey(deploy)
	stopper, stopChan := synchelpers.NewErrChanStopper(func(err error) {
		d.StopperHandlerService.Remove(objKey)
	})
	d.StopperHandlerService.Add(objKey, stopper)

	go func() {
		for {
			select {
			case e := <-wi:
				for _, handler := range eventHandler {
					handler.OnWatchEvent(e)
				}
			case err := <-stopChan:
				if err != nil {
					fmt.Println("Stopping watch of", deploy.Name, "pods in", deploy.Namespace)
				}
				return
			}
		}
	}()
	return nil
}
