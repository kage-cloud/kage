package kubeinformer

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

type KageMesh struct {
	InformerClient        kube.InformerClient            `inject:"InformerClient"`
	KubeClient            kube.Client                    `inject:"KubeClient"`
	KubeReaderService     service.KubeReaderService      `inject:"KubeReaderService"`
	EnvoyEndpointsService service.CanaryEndpointsService `inject:"CanaryEndpointsService"`
	KageMeshService       service.KageMeshService        `inject:"KageMeshService"`
	ProxyService          service.ProxyService           `inject:"ProxyService"`
}

func (k *KageMesh) StartAsync(ctx context.Context) error {
	informerSpec := kinformer.InformerSpec{
		NamespaceKind: ktypes.NamespaceKind{Kind: ktypes.KindService},
		BatchDuration: 5 * time.Second,
		Handlers: []kinformer.InformEventHandler{
			&kinformer.InformEventHandlerFuncs{
				OnWatch: func(event watch.Event) error {
					switch event.Type {
					case watch.Added, watch.Modified:
						if svc, ok := event.Object.(*corev1.Service); ok {
							_ = k.addService(svc)
						}
					case watch.Deleted, watch.Error:
						// if a service is removed, it could be under two states.
						// 1. it was forwarding all network calls to a kage proxy
						// 2. it wasn't
						// in both cases, the kage proxy wont care if it's no longer doing that.
					}
					return nil
				},
				OnList: func(li metav1.ListInterface) error {
					stream := kstream.StreamFromList(li)

					for _, obj := range stream.Collect().Objects() {
						svc, ok := obj.(*corev1.Service)
						if !ok {
							continue
						}
						if err := k.addService(svc); err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	}

	return k.InformerClient.Inform(ctx, informerSpec)
}

func (k *KageMesh) addService(svc *corev1.Service) error {
	opt := kconfig.Opt{Namespace: svc.Namespace}
	objs, err := k.KubeReaderService.List(labels.SelectorFromValidatedSet(svc.Spec.Selector), ktypes.KindPod, opt)
	if err != nil {
		return err
	}

	for _, obj := range kstream.StreamFromList(objs).Collect().Objects() {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			continue
		}

		kageProxyDeploys, err := k.listKageProxyDeploysForPod(pod)
		if err != nil {
			continue
		}

		for _, mesh := range kageProxyDeploys.Items {
			xdsAnno, err := k.KageMeshService.UnmarshalXdsMeta(&mesh)
			if err != nil {
				continue
			}

			logrus.WithField("name", svc.Name).
				WithField("namespace", svc.Namespace).
				Debug("Service routes to pod under a kage proxy.")
			_ = k.EnvoyEndpointsService.StorePod(pod)

			if xdsAnno.ServiceSelectors == nil {
				xdsAnno.ServiceSelectors = map[string]map[string]string{}
			}
			xdsAnno.ServiceSelectors[svc.Name] = ktypes.PodSelectorAsSet(svc)

			if _, err := k.KubeClient.Update(&mesh, opt); err != nil {
				logrus.WithError(err).
					WithField("name", mesh.Name).
					WithField("namespace", mesh.Namespace).
					WithField("service", svc.Name).
					Error("Failed to update mesh for service.")
				continue
			}

			if err := k.ProxyService.ProxyService(svc, meta.ToMap(&xdsAnno.Config.XdsId)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (k *KageMesh) listKageProxyDeploysForPod(pod *corev1.Pod) (*appsv1.DeploymentList, error) {
	opt := kconfig.Opt{Namespace: pod.Namespace}
	depLi, err := k.KubeReaderService.List(service.KageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(depLi).Filter(func(object metav1.Object) bool {
		xdsAnno, err := k.KageMeshService.UnmarshalXdsMeta(object)
		if err != nil {
			return false
		}
		return k.KageMeshService.TargetsPod(xdsAnno, pod)
	}).Collect().Deployments(), nil
}
