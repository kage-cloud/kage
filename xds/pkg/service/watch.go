package service

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kengine"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"time"
)

const WatchServiceKey = "WatchService"

type WatchService interface {
	Pods(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...kengine.InformEventHandler) error

	Services(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...kengine.InformEventHandler) error
	DeploymentPods(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error
	DeploymentServices(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error
	Deployment(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error
}

type watchService struct {
	KubeClient        kube.Client         `inject:"KubeClient"`
	InformerClient    kube.InformerClient `inject:"InformerClient"`
	KubeReaderService KubeReaderService   `inject:"KubeReaderService"`
}

func (w *watchService) Deployment(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error {
	spec := kengine.InformerSpec{
		NamespaceKind: ktypes.NewNamespaceKind(deploy.Namespace, ktypes.KindDeploy),
		BatchDuration: batchTime,
		Filter: func(object metav1.Object) bool {
			if v, ok := object.(*appsv1.Deployment); ok {
				return deploy.Name == v.Name && deploy.Namespace == v.Namespace
			}
			return false
		},
		Handlers: eventHandlers,
	}

	return w.InformerClient.Inform(ctx, spec)
}

func (w *watchService) Services(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...kengine.InformEventHandler) error {
	spec := kengine.InformerSpec{
		NamespaceKind: ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindService),
		BatchDuration: batchTime,
		Filter:        filter,
		Handlers:      eventHandlers,
	}

	return w.InformerClient.Inform(ctx, spec)
}

func (w *watchService) Pods(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...kengine.InformEventHandler) error {
	spec := kengine.InformerSpec{
		NamespaceKind: ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindPod),
		BatchDuration: batchTime,
		Filter:        filter,
		Handlers:      eventHandlers,
	}

	return w.InformerClient.Inform(ctx, spec)
}

func (w *watchService) DeploymentServices(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error {
	ns := kubeutil.DeploymentPodNamespace(deploy)

	spec := kengine.InformerSpec{
		NamespaceKind: ktypes.NewNamespaceKind(ns, ktypes.KindService),
		BatchDuration: batchTime,
		Filter: func(object metav1.Object) bool {
			if v, ok := object.(*corev1.Service); ok && v.Namespace == ns {
				selector := labels.SelectorFromSet(v.Spec.Selector)
				return selector.Matches(labels.Set(deploy.Spec.Template.Labels))
			}
			return false
		},
		Handlers: eventHandlers,
	}

	return w.InformerClient.Inform(ctx, spec)
}

func (w *watchService) DeploymentPods(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...kengine.InformEventHandler) error {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	ns := kubeutil.DeploymentPodNamespace(deploy)

	opt := kconfig.Opt{Namespace: ns}
	return w.Pods(ctx, kubeutil.SelectedObjectInNamespaceFilter(selector, opt), batchTime, opt, eventHandlers...)
}
