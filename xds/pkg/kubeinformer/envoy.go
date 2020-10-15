package kubeinformer

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const EnvoyKubeControllerKey = "EnvoyKubeController"

type Envoy struct {
	CanaryEndpointsService service.CanaryEndpointsService `inject:"CanaryEndpointsService"`
	InformerClient         kube.InformerClient            `inject:"InformerClient"`
}

func (e *Envoy) Inform(ctx context.Context) error {
	return e.InformerClient.Inform(ctx, kinformer.InformerSpec{
		NamespaceKind: ktypes.NamespaceKind{Kind: ktypes.KindPod},
		BatchDuration: 5 * time.Second,
		Handlers: []kinformer.InformEventHandler{
			e.kagePodEventHandler(),
		},
	})
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
				_ = e.CanaryEndpointsService.StorePod(pod)
			case watch.Error, watch.Deleted:
				_ = e.CanaryEndpointsService.RemovePod(pod)
			}
			return nil
		},
		OnList: func(li metav1.ListInterface) error {
			stream := kstream.StreamFromList(li)

			for _, v := range stream.Collect().Objects() {
				pod, ok := v.(*corev1.Pod)
				if !ok {
					continue
				}

				if err := e.CanaryEndpointsService.StorePod(pod); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
