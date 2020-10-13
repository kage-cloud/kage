package kubecontroller

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	EnvoyEndpointsService service.EnvoyEndpointsService `inject:"EnvoyEndpointsService"`
	KageMeshService       service.KageMeshService       `inject:"KageMeshService"`
	CanaryService         service.CanaryService         `inject:"CanaryService"`
	InformerClient        kube.InformerClient           `inject:"InformerClient"`
}

func (e *Envoy) StartAsync(ctx context.Context) error {
	informerSpec := kinformer.InformerSpec{
		BatchDuration: 15 * time.Second,
		Filter:        kfilter.LabelSelectorFilter(labels.SelectorFromValidatedSet(meta.ToMap(&meta.CanaryMarker{Canary: true}))),
		Handlers:      nil,
	}

	for _, controller := range controllers {
		informerSpec.NamespaceKind = ktypes.NamespaceKind{Kind: controller}
		if err := e.InformerClient.Inform(ctx, informerSpec); err != nil {
			return err
		}
	}

	return e.InformerClient.Inform(ctx, kinformer.InformerSpec{
		NamespaceKind: ktypes.NamespaceKind{Kind: ktypes.KindPod},
		BatchDuration: 5 * time.Second,
		Handlers: []kinformer.InformEventHandler{
			e.kagePodEventHandler(),
		},
	})
}

func (e *Envoy) canaryEventHandler() kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			switch event.Type {
			case watch.Error, watch.Deleted:
				e.KageMeshService.
			}
		},
	}
}

func (e *Envoy) kagePodEventHandler() kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				return nil
			}
			switch event.Type {
			case watch.Added, watch.Modified:
				_ = e.storeEnvoyEndpoint(pod)
			case watch.Error, watch.Deleted:
				_ = e.EnvoyEndpointsService.RemovePod(pod)
			}
			return nil
		},
		OnList: func(li metav1.ListInterface) error {
			stream := kstream.StreamFromList(li)

			for _, v := range stream.Objects() {
				pod, ok := v.(*corev1.Pod)
				if !ok {
					continue
				}

				if err := e.storeEnvoyEndpoint(pod); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func (e *Envoy) storeEnvoyEndpoint(pod *corev1.Pod) error {
	meshes, err := e.KageMeshService.ListXdsForPod(pod)
	if err != nil {
		return err
	}

	canaryAnno := e.CanaryService.FetchForPod(pod)
	if canaryAnno == nil {
		for _, mesh := range meshes {
			if err := e.EnvoyEndpointsService.StorePod(meta.SourceControllerType, &mesh, pod); err != nil {
				return err
			}
		}

		return nil
	}

	if len(meshes) == 0 {
		xdsAnno, err := e.KageMeshService.CreateFromCanary(canaryAnno)
		if err != nil {
			return err
		}
		meshes = append(meshes, *xdsAnno)
	}

	for _, mesh := range meshes {
		if err := e.EnvoyEndpointsService.StorePod(meta.CanaryControllerType, &mesh, pod); err != nil {
			return err
		}
	}

	return nil
}
