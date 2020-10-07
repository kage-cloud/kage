package kubecontroller

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/model/envoyepctlr"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const EnvoyKubeControllerKey = "EnvoyKubeController"

var controllers = []ktypes.Kind{
	ktypes.KindStatefulSet,
	ktypes.KindDaemonSet,
	ktypes.KindDeployment,
	ktypes.KindPod,
	ktypes.KindReplicaSet,
}

type Envoy struct {
	LockdownService       service.LockdownService       `inject:"LockdownService"`
	KubeReaderService     service.KubeReaderService     `inject:"KubeReaderService"`
	ListenerFactory       factory.ListenerFactory       `inject:"ListenerFactory"`
	EndpointFactory       factory.EndpointFactory       `inject:"EndpointFactory"`
	EnvoyInformerService  service.EnvoyInformerService  `inject:"EnvoyInformerService"`
	EnvoyEndpointsService service.EnvoyEndpointsService `inject:"EnvoyEndpointsService"`
	InformerClient        kube.InformerClient           `inject:"InformerClient"`
	StoreClient           snap.StoreClient              `inject:"StoreClient"`
	KubeClient            kube.Client                   `inject:"KubeClient"`
}

func (e *Envoy) StartAsync(ctx context.Context, spec *envoyepctlr.Spec) error {
	selector := labels.SelectorFromSet(meta.ToMap(&meta.Kage{Canary: true}))
	informerSpec := kinformer.InformerSpec{
		BatchDuration: 3 * time.Second,
		Filter:        kfilter.AnnotationSelectorFilter(selector),
		Handlers: []kinformer.InformEventHandler{
			e.handleUserAnnotation(),
		},
	}

	for _, v := range controllers {
		informerSpec.NamespaceKind = ktypes.NamespaceKind{Kind: v}
		if err := e.InformerClient.Inform(ctx, informerSpec); err != nil {
			return err
		}
	}

	return nil
}

// watch all pods who are owned by a controller with the marker annotations
func (e *Envoy) envoyEndpointHandler() kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			switch event.Type {
			case watch.Added, watch.Modified:
				if v, ok := event.Object.(*corev1.Pod); ok {
					_ = e.storePod(v)
				}
			case watch.Deleted:
				if v, ok := event.Object.(*corev1.Pod); ok {
					_ = e.EnvoyEndpointsService.RemovePod(v)
				}
			case watch.Error:
				switch typ := event.Object.(type) {
				case *corev1.Pod:
					_ = e.EnvoyEndpointsService.RemovePod(typ)
				}
			}
			return nil
		},
		OnList: func(li metav1.ListInterface) error {
			pods := li.(*corev1.PodList)
			for _, v := range pods.Items {
				_ = e.storePod(&v)
			}
			return nil
		},
	}
}

func (e *Envoy) storePod(pod *corev1.Pod) error {
	canaryAnno := e.getCanaryAnnoForPod(pod)
	controllerType := meta.SourceControllerType
	if canaryAnno != nil {
		controllerType = meta.CanaryControllerType
	}

	meshAnnos, err := e.findAllKageMeshesForPod(pod)
	if err != nil {
		return err
	}

	for _, v := range meshAnnos {
		_ = e.EnvoyEndpointsService.StorePod(controllerType, &v, pod)
	}

	return nil
}

func (e *Envoy) findAllKageMeshesForPod(pod *corev1.Pod) ([]meta.Xds, error) {
	opt := kconfig.Opt{Namespace: pod.Namespace}
	selector := labels.SelectorFromValidatedSet(meta.ToMap(&meta.MeshMarkerLabel{IsMesh: true}))
	meshes, err := e.KubeReaderService.List(selector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	meshStream := kstream.StreamFromList(meshes)

	xdsAnnos := make([]meta.Xds, 0, meshStream.Len())
	meshStream.ForEach(func(obj runtime.Object) {
		v, ok := obj.(metav1.Object)

		if !ok {
			return
		}

		xdsAnno := new(meta.Xds)
		err := meta.FromMap(v.GetAnnotations(), xdsAnno)
		if err != nil {
			return
		}

		if xdsAnno.LabelSelector != nil && labels.SelectorFromValidatedSet(xdsAnno.LabelSelector).Matches(labels.Set(pod.Labels)) {
			return
		}

		xdsAnnos = append(xdsAnnos, *xdsAnno)
	})

	return xdsAnnos, nil
}

func (e *Envoy) handleUserAnnotation() kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			switch event.Type {
			case watch.Added, watch.Modified:
				_ = e.storeCanaryAndSource(event.Object)
			case watch.Error, watch.Deleted:
				// remove the node id from the canary and unlock the service(s).
			}

			return nil
		},
		OnList: func(li metav1.ListInterface) error {
			stream := kstream.StreamFromList(li)
			for _, v := range stream.Objects() {
				return e.storeCanaryAndSource(v)
			}
			return nil
		},
	}
}

func (e *Envoy) storeCanaryAndSource(obj runtime.Object) error {
	if ktypes.IsController(ktypes.KindFromObject(obj)) {
		return nil
	}

	canary, ok := obj.(metav1.Object)
	if !ok {
		return nil
	}

	controllerAnnos := canary.GetAnnotations()
	if controllerAnnos == nil {
		return except.NewError("canary has no annotations", except.ErrInvalid)
	}

	canaryAnno := new(meta.Canary)
	if err := meta.FromMap(controllerAnnos, canaryAnno); err != nil {
		return except.NewError("invalid annotation for canary: %s", except.ErrInvalid, err.Error())
	}

	xdsAnno, err := e.EnvoyInformerService.GetOrInitXds(canary)
	if err != nil {
		return err
	}

	sourceOpt := kconfig.Opt{Namespace: canaryAnno.Source.Namespace}
	sourceKind := ktypes.Kind(canaryAnno.Source.Kind)

	err = e.getAndStoreControllerPods(canaryAnno.Source.Name, meta.SourceControllerType, xdsAnno, sourceKind, sourceOpt)
	if err != nil {
		return err
	}

	if err := e.storeControllerPods(obj, xdsAnno, meta.CanaryControllerType); err != nil {
		return err
	}

	return nil
}

func (e *Envoy) removeCanaryAndSource(obj runtime.Object) error {
	if ktypes.IsController(ktypes.KindFromObject(obj)) {
		return nil
	}

	canary, ok := obj.(metav1.Object)
	if !ok {
		return nil
	}

	controllerAnnos := canary.GetAnnotations()
	if controllerAnnos == nil {
		return except.NewError("canary has no annotations", except.ErrInvalid)
	}

	canaryAnno := new(meta.Canary)
	_ = meta.FromMap(controllerAnnos, canaryAnno)

	xdsAnno, err := e.EnvoyInformerService.GetOrInitXds(canary)
	if err != nil {
		return err
	}

	sourceOpt := kconfig.Opt{Namespace: canaryAnno.Source.Namespace}
	sourceKind := ktypes.Kind(canaryAnno.Source.Kind)

	err = e.getAndStoreControllerPods(canaryAnno.Source.Name, meta.SourceControllerType, xdsAnno, sourceKind, sourceOpt)
	if err != nil {
		return err
	}

	if err := e.storeControllerPods(obj, xdsAnno, meta.CanaryControllerType); err != nil {
		return err
	}

	return nil
}

func (e *Envoy) getAndStoreControllerPods(name string, controllerType meta.ControllerType, xdsAnno *meta.Xds, kind ktypes.Kind, opt kconfig.Opt) error {
	source, err := e.KubeReaderService.Get(name, kind, opt)
	if err != nil {
		return err
	}

	return e.storeControllerPods(source, xdsAnno, controllerType)
}

func (e *Envoy) storeControllerPods(source runtime.Object, xdsAnno *meta.Xds, controllerType meta.ControllerType) error {
	metaObj, ok := source.(metav1.Object)
	if !ok {
		return nil
	}
	opt := kconfig.Opt{Namespace: metaObj.GetNamespace()}
	sourceLi, err := e.KubeReaderService.List(ktypes.GetLabelSelector(source), ktypes.KindPod, opt)
	if err != nil {
		return err
	}

	for _, v := range kstream.StreamFromList(sourceLi).Objects() {
		if pod, ok := v.(*corev1.Pod); ok {
			_ = e.EnvoyEndpointsService.StorePod(controllerType, xdsAnno, pod)
		}
	}

	return nil
}

func (e *Envoy) getCanaryAnnoForPod(pod *corev1.Pod) *meta.Canary {
	var canaryAnno *meta.Canary
	_ = e.KubeReaderService.WalkControllers(pod, func(controller runtime.Object) (bool, error) {
		metaObj, ok := controller.(metav1.Object)
		if !ok {
			return true, nil
		}
		annos := metaObj.GetAnnotations()

		if err := meta.FromMap(annos, canaryAnno); err != nil {
			return true, nil
		}

		if canaryAnno.Source.Name != "" {
			return false, nil
		}

		return true, nil
	})

	return canaryAnno
}
