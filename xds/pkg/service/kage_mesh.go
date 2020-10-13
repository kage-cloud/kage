package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"strings"
	"time"
)

const KageMeshServiceKey = "KageMeshService"

var kageProxySelector = labels.SelectorFromValidatedSet(meta.ToMap(&meta.MeshMarker{IsMesh: true}))

type KageMeshService interface {
	CreateFromCanary(canary *meta.Canary) (*meta.Xds, error)
	Remove(xds *meta.Xds, opt kconfig.Opt) error
	ListXdsForPod(pod *corev1.Pod) ([]meta.Xds, error)
	Inform(ctx context.Context) error
}

type kageMeshService struct {
	KubeClient            kube.Client             `inject:"KubeClient"`
	KubeReaderService     KubeReaderService       `inject:"KubeReaderService"`
	KageMeshFactory       factory.KageMeshFactory `inject:"KageMeshFactory"`
	MeshConfigService     MeshConfigService       `inject:"MeshConfigService"`
	EnvoyEndpointsService EnvoyEndpointsService   `inject:"EnvoyEndpointsService"`
	InformerClient        kube.InformerClient     `inject:"InformerClient"`
	CanaryService         CanaryService           `inject:"CanaryService"`
	ProxyService          ProxyService            `inject:"ProxyService"`
}

func (k *kageMeshService) Remove(xds *meta.Xds, opt kconfig.Opt) error {
	dep, err := k.KubeReaderService.GetDeploy(xds.Name, opt)
	if err != nil {
		return err
	}

	proxiedSvcs, err := k.listProxiedServicesForDeploy(dep)
	if err != nil {
		return err
	}

	for _, v := range proxiedSvcs.Items {
		if err := k.removeForService(&v, opt); err != nil {
			logrus.WithError(err).
				WithField("name", v.Name).
				WithField("namespace", v.Namespace).
				Error("Failed to unlock service.")
			return err
		}
	}

	return nil
}

func (k *kageMeshService) ListXdsForPod(pod *corev1.Pod) ([]meta.Xds, error) {
	opt := kconfig.Opt{Namespace: pod.Namespace}
	meshes, err := k.listMeshDeploys(opt)
	if err != nil {
		return nil, err
	}

	return k.listXdsAnnosForPod(meshes, pod)
}

func (k *kageMeshService) Inform(ctx context.Context) error {
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

func (k *kageMeshService) CreateFromCanary(canary *meta.Canary) (*meta.Xds, error) {
	kageMeshName := k.genKageMeshName(&canary.SourceObj)
	opt := kconfig.Opt{Namespace: canary.SourceObj.Namespace}
	kageMeshDeploy, err := k.KubeReaderService.GetDeploy(kageMeshName, opt)
	var xdsAnno *meta.Xds
	if errors.IsNotFound(err) {
		kageMeshDeploy, xdsAnno, err = k.createKageMeshDeploy(kageMeshName, canary, opt)
		if err != nil {
			return nil, err
		}
	} else {
		xdsAnno, err = k.unmarshalXdsMeta(kageMeshDeploy)
		if err != nil {
			return nil, err
		}
	}

	return xdsAnno, nil
}

func (k *kageMeshService) removeForService(svc *corev1.Service, opt kconfig.Opt) error {
	meshes, err := k.listDeploysForProxiedService(svc)
	if err != nil {
		return err
	}

	if len(meshes.Items) <= 1 {
		if err := k.ProxyService.ReleaseService(svc, opt); err != nil {
			return err
		}
	}
	return nil
}

func (k *kageMeshService) listDeploysForProxiedService(svc *corev1.Service) (*appsv1.DeploymentList, error) {
	opt := kconfig.Opt{Namespace: svc.Namespace}
	meshes, err := k.KubeReaderService.List(kageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(meshes).Filter(func(object metav1.Object) bool {
		xdsAnno, err := k.unmarshalXdsMeta(object)
		if err != nil {
			return false
		}

		return xdsAnno.ProxiedServices[svc.Name]
	}).Collect().Deployments(), nil
}

func (k *kageMeshService) listProxiedServicesForDeploy(dep *appsv1.Deployment) (*corev1.ServiceList, error) {
	opt := kconfig.Opt{Namespace: dep.Namespace}
	xdsAnno := new(meta.Xds)
	if err := meta.FromMap(dep.Annotations, xdsAnno); err != nil {
		return nil, err
	}

	svcs := make([]corev1.Service, 0, len(xdsAnno.ProxiedServices))
	for name := range xdsAnno.ProxiedServices {
		obj, err := k.KubeReaderService.Get(name, ktypes.KindService, opt)
		if err != nil {
			continue
		}

		svcs = append(svcs, *obj.(*corev1.Service))
	}

	return &corev1.ServiceList{Items: svcs}, nil
}

func (k *kageMeshService) listMeshDeploys(opt kconfig.Opt) (*appsv1.DeploymentList, error) {
	meshes, err := k.KubeReaderService.List(kageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(meshes).Collect().Deployments(), nil
}

func (k *kageMeshService) listXdsAnnosForPod(meshes *appsv1.DeploymentList, pod *corev1.Pod) ([]meta.Xds, error) {
	xdsAnnos := make([]meta.Xds, 0, len(meshes.Items))

	for _, v := range meshes.Items {
		xdsAnno, err := k.unmarshalXdsMeta(&v)
		if err != nil {
			continue
		}

		if k.targetsPod(xdsAnno, pod) {
			xdsAnnos = append(xdsAnnos, *xdsAnno)
		}
	}

	return xdsAnnos, nil
}

func (k *kageMeshService) targetsPod(xdsAnno *meta.Xds, pod *corev1.Pod) bool {
	return xdsAnno.LabelSelector != nil && labels.SelectorFromValidatedSet(xdsAnno.LabelSelector).Matches(labels.Set(pod.Labels))
}

func (k *kageMeshService) listDeploysForPod(pod *corev1.Pod) (*appsv1.DeploymentList, error) {
	opt := kconfig.Opt{Namespace: pod.Namespace}
	depLi, err := k.KubeReaderService.List(kageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(depLi).Filter(func(object metav1.Object) bool {
		xdsAnno, err := k.unmarshalXdsMeta(object)
		if err != nil {
			return false
		}
		return k.targetsPod(xdsAnno, pod)
	}).Collect().Deployments(), nil
}

func (k *kageMeshService) addService(svc *corev1.Service) error {
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

		meshDeps, err := k.listDeploysForPod(pod)
		if err != nil {
			continue
		}

		for _, mesh := range meshDeps.Items {
			canary := k.CanaryService.FetchForPod(pod)
			controllerType := meta.SourceControllerType
			if canary != nil {
				controllerType = meta.CanaryControllerType
			}

			xdsAnno, err := k.unmarshalXdsMeta(&mesh)
			if err != nil {
				continue
			}

			logrus.WithField("name", svc.Name).
				WithField("namespace", svc.Namespace).
				Debug("Service routes to pod under a kage proxy.")
			_ = k.EnvoyEndpointsService.StorePod(controllerType, xdsAnno, pod)

			if xdsAnno.ProxiedServices == nil {
				xdsAnno.ProxiedServices = map[string]bool{}
			}

			xdsAnno.ProxiedServices[svc.Name] = true

			xdsAnno.LabelSelector = ktypes.UnionSet(xdsAnno.LabelSelector, svc.Spec.Selector)

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

func (k *kageMeshService) createKageMeshDeploy(name string, canary *meta.Canary, opt kconfig.Opt) (*appsv1.Deployment, *meta.Xds, error) {
	xdsAnno := &meta.Xds{
		Name:          name,
		LabelSelector: map[string]string{},
		Canary:        *canary,
		Config: meta.XdsConfig{
			XdsId: meta.XdsId{NodeId: uuid.New().String()},
			Canary: meta.EnvoyConfig{
				ClusterName: "canary",
			},
			Source: meta.EnvoyConfig{
				ClusterName: "source",
			},
		},
	}

	baseline, err := k.MeshConfigService.FromXdsConfig(&xdsAnno.Config)
	if err != nil {
		return nil, nil, err
	}

	cm := k.KageMeshFactory.BaselineConfigMap(name, baseline)
	_, err = k.KubeClient.Create(cm, opt)
	if err != nil {
		return nil, nil, err
	}

	dep := k.KageMeshFactory.Deploy(name, &xdsAnno.Config)
	dep, err = k.KubeClient.CreateDeploy(dep, opt)
	if err != nil {
		return nil, nil, err
	}

	return dep, xdsAnno, nil
}

func (k *kageMeshService) marshalXdsMeta(obj metav1.Object, xds *meta.Xds) {
	obj.SetAnnotations(meta.Merge(obj.GetAnnotations(), xds))
	obj.SetLabels(meta.Merge(obj.GetLabels(), &xds.Config.XdsId))
}

func (k *kageMeshService) unmarshalXdsMeta(obj metav1.Object) (*meta.Xds, error) {
	xdsAnno := new(meta.Xds)
	if err := meta.FromMap(obj.GetAnnotations(), xdsAnno); err != nil {
		return nil, err
	}

	if err := meta.FromMap(obj.GetLabels(), xdsAnno); err != nil {
		return nil, err
	}

	return xdsAnno, nil
}

func (k *kageMeshService) genKageMeshName(source *meta.ObjRef) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-kage-mesh", source.Name, source.Kind))
}
