package service

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/kube/kubeutil"
	"github.com/kage-cloud/kage/synchelpers"
	"github.com/kage-cloud/kage/xds/model"
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
	Pods(selector labels.Selector, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error

	// Watch services who's selectors match the specified labels
	Services(ls map[string]string, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error
	DeploymentPods(deploy *appsv1.Deployment, batchTime time.Duration, eventHandler ...model.InformEventHandler) error
	DeploymentServices(deploy *appsv1.Deployment, batchTime time.Duration, eventHandler ...model.InformEventHandler) error
}

type watchService struct {
	KubeClient kube.Client `inject:"KubeClient"`
	StopperMap synchelpers.StopperMap
}

func (w *watchService) Services(ls map[string]string, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error {
	svcs, wi := w.KubeClient.InformAndListServices(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			return object.GetNamespace() == opt.Namespace && labels.SelectorFromSet(v.Spec.Selector).Matches(labels.Set(ls))
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

	objKey := labels.Set(ls).String() + opt.Namespace
	stopper := synchelpers.NewStopper(func(err error) {
		w.StopperMap.Remove(objKey)
	})

	w.createQueue(stopper, batchTime, wi, eventHandlers...)

	w.StopperMap.Add(objKey, stopper)

	return nil
}

func (w *watchService) Pods(selector labels.Selector, batchTime time.Duration, opt kconfig.Opt, eventHandlers ...model.InformEventHandler) error {
	pods, wi := w.KubeClient.InformAndListPod(func(object metav1.Object) bool {
		return object.GetNamespace() == opt.Namespace && selector.Matches(labels.Set(object.GetLabels()))
	})
	podList := &corev1.PodList{
		Items: pods,
	}

	for _, eh := range eventHandlers {
		if err := eh.OnListEvent(podList); err != nil {
			return err
		}
	}

	objKey := selector.String() + opt.Namespace
	stopper := synchelpers.NewStopper(func(err error) {
		w.StopperMap.Remove(objKey)
	})

	w.createQueue(stopper, batchTime, wi, eventHandlers...)

	w.StopperMap.Add(objKey, stopper)

	return nil
}

func (w *watchService) DeploymentServices(deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error {
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

	objKey := kubeutil.ObjectKey(deploy)
	stopper := synchelpers.NewStopper(func(err error) {
		w.StopperMap.Remove(objKey)
	})

	w.createQueue(stopper, batchTime, wi, eventHandlers...)

	w.StopperMap.Add(objKey, stopper)
	return nil
}

func (w *watchService) DeploymentPods(deploy *appsv1.Deployment, batchTime time.Duration, eventHandlers ...model.InformEventHandler) error {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return err
	}

	selector.String()
	ns := kubeutil.DeploymentPodNamespace(deploy)

	return w.Pods(selector, batchTime, kconfig.Opt{Namespace: ns}, eventHandlers...)
}

func (w *watchService) handleItemLoop(stopper synchelpers.Stopper, queue workqueue.Interface, handlers ...model.InformEventHandler) {
	shutdown := false
	loopTimer := time.NewTimer(time.Second)
	for !shutdown {
		<-loopTimer.C

		var item interface{}
		item, shutdown = queue.Get()
		if v, ok := item.(watch.Event); ok {
			for _, h := range handlers {
				if !h.OnWatchEvent(v) {
					stopper.Stop(nil)
				}
			}
		}
		if !shutdown {
			shutdown = stopper.IsStopped()
		}
	}
}

func (w *watchService) createQueue(stopper synchelpers.Stopper, batchTime time.Duration, watchChan <-chan watch.Event, handlers ...model.InformEventHandler) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	go w.handleItemLoop(stopper, queue, handlers...)
	go w.queueItemLoop(stopper, batchTime, queue, watchChan)
}

func (w *watchService) queueItemLoop(stopper synchelpers.Stopper, batchTime time.Duration, queue workqueue.RateLimitingInterface, watchChan <-chan watch.Event) {
	loopTimer := time.NewTimer(time.Second)
	for !stopper.IsStopped() {
		<-loopTimer.C

		e := <-watchChan
		queue.AddAfter(e, batchTime)
	}
}

func deployWatchServiceFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&watchService{
		StopperMap: synchelpers.NewStopperMap(),
	})
}
