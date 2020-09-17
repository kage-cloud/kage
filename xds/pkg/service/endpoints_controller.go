package service

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/ktypes/objconv"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kfilter"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kinformer"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const EndpointsControllerServiceKey = "EndpointsControllerService"

type EndpointsControllerService interface {
	StartForDeploys(ctx context.Context, blacklist []appsv1.Deployment, opt kconfig.Opt) error
	Stop(blacklist []appsv1.Deployment, opt kconfig.Opt) error
}

type endpointsControllerService struct {
	LockdownService   LockdownService   `inject:"LockdownService"`
	WatchService      WatchService      `inject:"WatchService"`
	XdsService        XdsService        `inject:"XdsService"`
	KubeReaderService KubeReaderService `inject:"KubeReaderService"`
	KubeClient        kube.Client       `inject:"KubeClient"`
}

func (e *endpointsControllerService) Stop(blacklist []appsv1.Deployment, opt kconfig.Opt) error {
	allSvcs, err := e.KubeReaderService.ListServices(labels.Everything(), opt)
	if err != nil {
		return err
	}

	svcs := make([]corev1.Service, 0)
	for _, svc := range allSvcs {
		selectorSet, err := e.getServiceSelectorSet(&svc)
		if err != nil {
			continue
		}

		selector := selectorSet.AsSelectorPreValidated()

		for _, b := range blacklist {
			if selector.Matches(labels.Set(b.Spec.Template.Labels)) {
				svcs = append(svcs, svc)
				break
			}
		}
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

func (e *endpointsControllerService) StartForDeploys(ctx context.Context, blacklist []appsv1.Deployment, opt kconfig.Opt) error {
	err := e.WatchService.Services(ctx, e.watchFilter(blacklist, opt), time.Second, opt, &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				return except.NewError("Event did not contain service", except.ErrInvalid)
			}
			switch event.Type {
			case watch.Deleted:
				if err := e.KubeClient.Api().CoreV1().Endpoints(opt.Namespace).Delete(svc.Name, &metav1.DeleteOptions{}); err != nil {
					log.WithError(err).
						WithField("name", svc.Name).
						WithField("namespace", opt.Namespace).
						Error("Failed to delete endpoint.")
					return err
				}
				return nil
			case watch.Added, watch.Modified:
				rses, err := e.KubeReaderService.GetDeployReplicaSets(blacklist, opt)
				if err != nil {
					log.WithError(err).
						WithField("namespace", opt.Namespace).
						WithField("name", svc.Name).
						WithField("deploys", kubeutil.ObjNames(objconv.FromDeployments(blacklist))).
						Debug("Failed to get replicasets.")
					return err
				}
				if err := e.syncService(svc, rses, opt); err != nil {
					log.WithError(err).
						WithField("namespace", opt.Namespace).
						WithField("name", svc.Name).
						Error("Failed to sync service after it was updated.")
					return err
				}
			}
			return nil
		},
		OnList: func(obj runtime.Object) error {
			svcList, ok := obj.(*corev1.ServiceList)
			if !ok {
				return except.NewError("a service list was not returned on the watcher: %v", except.ErrInternalError, svcList)
			}

			selectors := make([]labels.Selector, 0, len(svcList.Items))
			for _, s := range svcList.Items {
				sel, err := e.getServiceSelectorSet(&s)
				if err != nil {
					continue
				}
				selectors = append(selectors, sel.AsSelectorPreValidated())

				if err := e.LockdownService.LockdownService(&s, opt); err != nil {
					return err
				}
			}

			filter := kfilter.LazyMatchesSelectorsFilter(selectors...)

			err := e.WatchService.Pods(ctx, filter, 3*time.Second, opt, &kinformer.InformEventHandlerFuncs{
				OnWatch: func(event watch.Event) error {

					switch event.Type {
					case watch.Modified, watch.Added, watch.Deleted:
						rses, err := e.KubeReaderService.GetDeployReplicaSets(blacklist, opt)
						if err != nil {
							log.WithError(err).
								WithField("namespace", opt.Namespace).
								WithField("deploys", kubeutil.ObjNames(objconv.FromDeployments(blacklist))).
								WithField("pod", event.Object.(metav1.Object).GetName()).
								Debug("Failed to find replicasets for deploys after pod was updated")
							return err
						}
						for _, s := range svcList.Items {
							if err := e.syncService(&s, rses, opt); err != nil {
								log.WithError(err).
									WithField("namespace", opt.Namespace).
									WithField("service", s.Name).
									WithField("pod", event.Object.(metav1.Object).GetName()).
									Error("Failed to sync service after pod was updated.")
								return err
							}
						}
					}
					return nil
				},
				OnList: func(obj runtime.Object) error {
					rses, err := e.KubeReaderService.GetDeployReplicaSets(blacklist, opt)
					if err != nil {
						return err
					}
					for _, s := range svcList.Items {
						if err := e.syncService(&s, rses, opt); err != nil {
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
		},
	})
	if err != nil {
		return err
	}

	return nil
}
func (e *endpointsControllerService) syncService(svc *corev1.Service, replicaSets []appsv1.ReplicaSet, opt kconfig.Opt) error {
	svcSet, err := e.getServiceSelectorSet(svc)
	if err != nil {
		return err
	}

	svcSelector := svcSet.AsSelectorPreValidated()

	pods, err := e.KubeReaderService.ListPods(svcSelector, opt)
	if err != nil {
		return err
	}

	ep, err := e.KubeReaderService.GetEndpoints(svc.Name, opt)
	if err != nil {
		return err
	}

	subsets := make([]corev1.EndpointSubset, 0)

	rsObjs := objconv.FromReplicaSets(replicaSets)

	for _, p := range pods {
		if kubeutil.IsOwned(&p, rsObjs...) {
			continue
		}

		ea := kubeutil.PodToEndpointAddress(&p)

		if ea.IP == "" {
			continue
		}

		addr := []corev1.EndpointAddress{*ea}

		for _, sp := range svc.Spec.Ports {
			port, err := kubeutil.FindPort(&p, &sp)
			if err != nil {
				log.WithError(err).
					WithField("namespace", opt.Namespace).
					WithField("service", svc.Name).
					WithField("pod", p.Name).
					Debug("Failed to find targeted service port for pod")
				continue
			}

			epp := kubeutil.EndpointPortFromServicePort(&sp, port)

			ss := corev1.EndpointSubset{
				Ports: []corev1.EndpointPort{*epp},
			}

			if p.Status.Phase == corev1.PodRunning {
				ss.Addresses = addr
			} else {
				ss.NotReadyAddresses = addr
			}
			subsets = append(subsets, ss)
		}
	}

	if len(subsets) > 0 {
		ep.Subsets = kubeutil.RepackSubsets(subsets)
		if _, err := e.KubeClient.SaveEndpoints(ep, opt); err != nil {
			log.WithError(err).
				WithField("endpoint", svc.Name).
				WithField("namespace", opt.Namespace).
				Error("Failed to update endpoint")
			return err
		}
		log.WithField("endpoint", svc.Name).
			WithField("namespace", opt.Namespace).
			WithField("addresses", kubeutil.Addresses(ep.Subsets)).
			Trace("Updated endpoint")
	} else {
		log.WithField("endpoint", svc.Name).
			WithField("namespace", opt.Namespace).
			Debug("No valid pods found. No update made to endpoint.")
	}

	return nil
}

func (e *endpointsControllerService) watchFilter(targets []appsv1.Deployment, opt kconfig.Opt) kfilter.Filter {
	return func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			selectorSet, err := e.getServiceSelectorSet(v)
			if err != nil {
				return false
			}
			selector := selectorSet.AsSelectorPreValidated()

			for _, t := range targets {
				if object.GetNamespace() == opt.Namespace && selector.Matches(labels.Set(t.Spec.Template.Labels)) {
					return true
				}
			}
		}
		return false
	}
}

func (e *endpointsControllerService) getServiceSelectorSet(svc *corev1.Service) (labels.Set, error) {
	sel := svc.Spec.Selector
	if sel == nil {
		ld, err := e.LockdownService.GetLockDown(svc)
		if err == nil {
			sel = ld.DeletedSelector
		}
	}
	return sel, nil
}
