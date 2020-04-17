package service

import (
	"context"
	"fmt"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/pkg/model"
	log "github.com/sirupsen/logrus"
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
			log.WithField("name", s.Name).
				WithField("namespace", s.Namespace).
				WithError(err).
				Error("Failed to release service from lockdown.")
		}
	}

	return nil
}

func (e *endpointsControllerService) StartWithBlacklistedEndpoints(ctx context.Context, blacklist labels.Set, opt kconfig.Opt) error {
	blacklistSelector := blacklist.AsSelectorPreValidated()
	err := e.WatchService.Services(ctx, e.watchFilter(blacklist, opt), time.Second, opt, &model.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) bool {
			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				return ok
			}
			switch event.Type {
			case watch.Deleted:
				kapi := e.KubeClient.Api()
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
	svcSet, err := e.getServiceSelector(svc)
	if err != nil {
		return err
	}

	svcSelector := svcSet.AsSelectorPreValidated()

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

func (e *endpointsControllerService) watchFilter(blacklist labels.Set, opt kconfig.Opt) kube.Filter {
	return func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			sel, err := e.getServiceSelector(v)
			if err != nil {
				return false
			}
			filter := object.GetNamespace() == opt.Namespace && v.Spec.Selector != nil && labels.SelectorFromSet(sel).Matches(labels.Set(blacklist))
			return filter
		}
		return false
	}
}

func (e *endpointsControllerService) getServiceSelector(svc *corev1.Service) (labels.Set, error) {
	sel := svc.Spec.Selector
	if sel == nil {
		ld, err := e.LockdownService.GetLockDown(svc)
		if err != nil {
			return nil, err
		}
		sel = ld.DeletedSelector
	}
	return sel, nil
}
