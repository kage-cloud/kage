package service

import (
	"context"
	"fmt"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/except"
	"github.com/kage-cloud/kage/xds/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const EndpointsControllerServiceKey = "EndpointsControllerService"

type EndpointsControllerService interface {
	StartWithBlacklistedEndpoints(ctx context.Context, blacklist labels.Set, opt kconfig.Opt) error
	Stop(blacklist labels.Set, opt kconfig.Opt) error
}

type endpointsControllerService struct {
	LockdownService LockdownService `inject:"LockdownService"`
	WatchService    WatchService    `inject:"WatchService"`
	XdsService      XdsService      `inject:"XdsService"`
	KubeClient      kube.Client     `inject:"KubeClient"`
}

func (e *endpointsControllerService) Stop(blacklist labels.Set, opt kconfig.Opt) error {
	svcs, err := e.KubeClient.ListServices(blacklist.AsSelectorPreValidated(), opt)
	if err != nil {
		return err
	}
	for _, s := range svcs {
		if err := e.LockdownService.ReleaseService(&s, opt); err != nil {
			return err
		}
	}

	return nil
}

func (e *endpointsControllerService) StartWithBlacklistedEndpoints(ctx context.Context, blacklist labels.Set, opt kconfig.Opt) error {
	blacklistSelector := blacklist.AsSelectorPreValidated()
	err := e.WatchService.Services(ctx, blacklist, time.Second, opt, &model.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) bool {
			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				return ok
			}
			switch event.Type {
			case watch.Deleted:
				kapi, err := e.KubeClient.Api(opt.Context)
				if err != nil {
					fmt.Println("failed to get the kube client api for context", opt.Context, ":", err.Error())
					return false
				}
				if err := kapi.CoreV1().Endpoints(opt.Namespace).Delete(svc.Name, &metav1.DeleteOptions{}); err != nil {
					fmt.Println("failed to delete endpoint", svc.Name, "in", opt.Namespace, ":", err.Error())
					return false
				}
				return false
			case watch.Added, watch.Modified:
				if err := e.syncService(svc, blacklistSelector, opt); err != nil {
					fmt.Println("failed to sync service", svc.Name, "in", opt.Namespace, ":", err.Error())
				}
			}
			return true
		},
		OnList: func(obj runtime.Object) error {
			svcList, ok := obj.(*corev1.ServiceList)
			if !ok {
				return except.NewError("a service list was not returned on the watcher: %v", except.ErrInternalError, svcList)
			}
			err := e.WatchService.Pods(ctx, blacklistSelector, 3*time.Second, opt, &model.InformEventHandlerFuncs{
				OnWatch: func(event watch.Event) bool {
					switch event.Type {
					case watch.Modified, watch.Added, watch.Deleted:
						for _, s := range svcList.Items {
							if err := e.syncService(&s, blacklistSelector, opt); err != nil {
								fmt.Println("failed to sync service", s.Name, "in", opt.Namespace, ":", err.Error())
							}
						}
					}
					return true
				},
				OnList: func(obj runtime.Object) error {
					for _, s := range svcList.Items {
						if err := e.syncService(&s, blacklistSelector, opt); err != nil {
							return err
						}
					}
					return nil
				},
			})

			if err != nil {
				return err
			}

			for _, s := range svcList.Items {
				if err := e.LockdownService.LockdownService(&s, opt); err != nil {
					return err
				}
			}
			return nil
		},
	})
	if err != nil {
		return err
	}

	return nil
}
func (e *endpointsControllerService) syncService(svc *corev1.Service, blackList labels.Selector, opt kconfig.Opt) error {
	if svc.Spec.Selector == nil {
		return nil
	}

	svcSelector := labels.SelectorFromSet(svc.Spec.Selector)
	pods, err := e.KubeClient.ListPods(svcSelector, opt)
	if err != nil {
		return err
	}

	ep, err := e.KubeClient.GetEndpoints(svc.Name, opt)
	if err != nil {
		return err
	}

	for _, p := range pods {
		if blackList.Matches(labels.Set(p.Labels)) {
			continue
		}

		ea := kubeutil.PodToEndpointAddress(&p)
		addr := []corev1.EndpointAddress{*ea}

		for _, sp := range svc.Spec.Ports {
			port, err := kubeutil.FindPort(&p, &sp)
			if err != nil {
				fmt.Println("Failed to find targeted service port for pod", p.Name, "and service", svc.Name, "in", opt.Namespace, ":", err.Error())
				continue
			}

			epp := kubeutil.EndpointPortFromServicePort(&sp, port)

			ep.Subsets = append(ep.Subsets, corev1.EndpointSubset{
				Addresses: addr,
				Ports:     []corev1.EndpointPort{*epp},
			})
		}
	}

	ep.Subsets = kubeutil.RepackSubsets(ep.Subsets)

	if _, err := e.KubeClient.UpdateEndpoints(ep, opt); err != nil {
		fmt.Println("Failed to update endpoint", svc.Name, "in", opt.Namespace, ":", err.Error())
		return err
	}

	return nil
}
