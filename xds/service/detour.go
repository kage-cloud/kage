package service

import (
	"fmt"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type DetourService interface {
	EndpointsDetour(from *appsv1.Deployment, to *appsv1.Deployment) error
}

type detourService struct {
	LockdownService    LockdownService    `inject:"LockdownService"`
	DeployWatchService DeployWatchService `inject:"DeployWatchService"`
	KubeClient         kube.Client        `inject:"KubeClient"`
}

func (d *detourService) EndpointsDetour(from *appsv1.Deployment, to *appsv1.Deployment) error {
	if err := d.DeployWatchService.DeploymentPods(from, &model.InformEventHandlerFuncs{
		OnWatch: nil,
		OnList:  nil,
	}); err != nil {
		return err
	}

	ns := kubeutil.DeploymentPodNamespace(from)

	opt := kconfig.Opt{Namespace: ns}

	svcs, err := d.KubeClient.ListServices("", opt)
	if err != nil {
		return err
	}

	svcs = kubeutil.MatchedServices(from.Spec.Template.Labels, svcs)

	for _, svc := range svcs {
		err = d.DeployWatchService.DeploymentPods(from, &model.InformEventHandlerFuncs{
			OnWatch: d.endpointsDetourOnWatch(&svc, opt),
			OnList:  d.endpointsDetourOnList(&svc, opt),
		})
		if err != nil {
			fmt.Println("Failed to lockdown service", svc.Name, ":", err.Error())
		}

		if err := d.LockdownService.LockdownService(&svc, opt); err != nil {
			fmt.Println("Failed to lockdown service", svc.Name, ":", err.Error())
			continue
		}
	}

	return nil
}

func (d *detourService) endpointsDetourOnList(service *corev1.Service, opt kconfig.Opt) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		//if v, ok := obj.(*corev1.Pod); ok {
		//	ep, err := d.KubeClient.GetEndpoints(service.Name, opt)
		//	if err != nil {
		//		fmt.Println("Failed to get endpoint", service.Name, "in", opt.Namespace, ":", err.Error())
		//		return err
		//	}
		//
		//	//for _, subset := range ep.Subsets {
		//	//	//for _, p := range subset.Ports {
		//	//	}
		//	//}
		//
		//	if _, err := d.KubeClient.UpdateEndpoints(ep, opt); err != nil {
		//		fmt.Println("Failed to update endpoint", service.Name, "in", opt.Namespace, ":", err.Error())
		//	}
		//}
		return nil
	}
}

func (d *detourService) endpointsDetourOnWatch(service *corev1.Service, opt kconfig.Opt) model.OnWatchEventFunc {
	return func(event watch.Event) {
		//if v, ok := obj.(*corev1.Endpoints); ok {
		//	for _, subset := range v.Subsets {
		//		for _, port := range subset.Ports {
		//		}
		//	}
		//}
	}
}
