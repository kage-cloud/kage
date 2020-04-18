package service

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kengine"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/pkg/model"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
	"time"
)

const WatchServiceKey = "WatchService"

type WatchService interface {
	Pods(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error

	Services(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error
	DeploymentPods(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error
	DeploymentServices(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error
	Deployment(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error
}

type watchService struct {
	KubeClient kube.Client `inject:"KubeClient"`
}

func (w *watchService) Deployment(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error {
	deploys, wi := w.KubeClient.InformAndListDeploys(func(object metav1.Object) bool {
		if v, ok := object.(*appsv1.Deployment); ok {
			return deploy.Name == v.Name && deploy.Namespace == v.Namespace
		}
		return false
	})

	vList := &appsv1.DeploymentList{
		Items: deploys,
	}

	for _, eh := range eventHandlers {
		if err := eh.OnListEvent(vList); err != nil {
			return err
		}
	}

	w.createQueue(ctx, batchTime, wi, eventHandlers...)

	return nil
}

func (w *watchService) Services(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error {
	svcs, wi := w.KubeClient.InformAndListServices(filter)

	svcList := &corev1.ServiceList{
		Items: svcs,
	}

	for _, eh := range eventHandlers {
		if err := eh.OnListEvent(svcList); err != nil {
			return err
		}
	}

	w.createQueue(ctx, batchTime, wi, eventHandlers...)

	return nil
}

func (w *watchService) Pods(ctx context.Context, filter kengine.Filter, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error {
	pods, wi := w.KubeClient.InformAndListPod(filter)
	podList := &corev1.PodList{
		Items: pods,
	}

	for _, eh := range eventHandlers {
		if err := eh.OnListEvent(podList); err != nil {
			return err
		}
	}

	w.createQueue(ctx, batchTime, wi, eventHandlers...)

	return nil
}

func (w *watchService) DeploymentServices(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error {
	ns := kubeutil.DeploymentPodNamespace(deploy)

	svcs, wi := w.KubeClient.InformAndListServices(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok && v.Namespace == ns {
			selector := labels.SelectorFromSet(v.Spec.Selector)
			return selector.Matches(labels.Set(deploy.Spec.Template.Labels))
		}
		return false
	})
	svcList := &corev1.ServiceList{
		Items: svcs,
	}

	for _, eh := range eventHandlers {
		if err := eh.OnListEvent(svcList); err != nil {
			return err
		}
	}

	w.createQueue(ctx, batchTime, wi, eventHandlers...)

	return nil
}

func (w *watchService) DeploymentPods(ctx context.Context, deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	ns := kubeutil.DeploymentPodNamespace(deploy)

	opt := kconfig.Opt{Namespace: ns}
	return w.Pods(ctx, kubeutil.SelectedObjectFilter(selector, opt), batchTime, opt, eventHandlers...)
}

func (w *watchService) handleItemLoop(ctx context.Context, queue workqueue.Interface, handlers ...model.InformEventHandler) {
	shutdown := false
	loopTicker := time.NewTicker(time.Second)
	for !shutdown {
		select {
		case <-loopTicker.C:
			var item interface{}
			item, shutdown = queue.Get()
			var isDone bool
			if v, ok := item.(watch.Event); ok {
				for _, h := range handlers {
					err, shouldContinue := h.OnWatchEvent(v)
					if !shouldContinue {
						shutdown = true
					}
					isDone = isDone && err == nil
				}
			}
			if isDone {
				queue.Done(item)
			}
		case <-ctx.Done():
			shutdown = true
		}
	}

	log.Debug("Exiting item handler loop.")
}

func (w *watchService) createQueue(ctx context.Context, batchTime time.Duration, watchChan <-chan watch.Event, handlers ...model.InformEventHandler) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	go w.handleItemLoop(ctx, queue, handlers...)
	go w.queueItemLoop(ctx, batchTime, queue, watchChan)
}

func (w *watchService) queueItemLoop(ctx context.Context, batchTime time.Duration, queue workqueue.RateLimitingInterface, watchChan <-chan watch.Event) {
	for !queue.ShuttingDown() {
		select {
		case e := <-watchChan:
			queue.AddAfter(e, batchTime)
		case <-ctx.Done():
			queue.ShutDown()
			return
		}
	}
}
