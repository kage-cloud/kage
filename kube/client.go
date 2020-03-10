package kube

import (
	"fmt"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/except"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

type Client interface {
	InformEndpoints(filter Filter) <-chan watch.Event
	InformDeploy(filter Filter) <-chan watch.Event
	WatchDeploy(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error)
	WaitTillDeployReady(name string, timeout time.Duration, opt kconfig.Opt) error
	GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error)
	ListConfigMaps(lo metav1.ListOptions, opt kconfig.Opt) ([]corev1.ConfigMap, error)
	DeleteConfigMap(name string, opt kconfig.Opt) error
	UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error)
	UpsertDeploy(dep *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error)
	DeleteDeploy(name string, opt kconfig.Opt) error
	GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error)
	GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error)
	GetService(name string, opt kconfig.Opt) (*corev1.Service, error)
	UpdateService(service *corev1.Service, opt kconfig.Opt) (*corev1.Service, error)
	ListServices(lo metav1.ListOptions, opt kconfig.Opt) ([]corev1.Service, error)
}

func NewClient() (Client, error) {
	conf, err := kconfig.NewConfigClient()
	if err != nil {
		return nil, err
	}

	apiClient, err := conf.Api("")
	if err != nil {
		return nil, err
	}

	fact := informers.NewSharedInformerFactoryWithOptions(apiClient, 0)

	return &client{
		Config:                     conf,
		SharedInformerFactory:      fact,
		informerHandlersByResource: map[string]chan struct{}{},
		mapLock:                    sync.RWMutex{},
	}, nil
}

type client struct {
	Config                     kconfig.Config
	SharedInformerFactory      informers.SharedInformerFactory
	informerHandlersByResource map[string]chan struct{}
	mapLock                    sync.RWMutex
}

func (c *client) InformEndpoints(filter Filter) <-chan watch.Event {
	ch, handler := c.handlerFactory(filter, func(obj interface{}) (object runtime.Object, b bool) {
		object, b = obj.(*corev1.Endpoints)
		return
	})
	c.SharedInformerFactory.Core().V1().Endpoints().Informer().AddEventHandler(handler)
	c.initInformer("endpoints", c.SharedInformerFactory.Core().V1().Endpoints().Informer())
	return ch
}

func (c *client) InformDeploy(filter Filter) <-chan watch.Event {
	ch, handler := c.handlerFactory(filter, func(obj interface{}) (object runtime.Object, b bool) {
		object, b = obj.(*appsv1.Deployment)
		return
	})
	c.SharedInformerFactory.Apps().V1().Deployments().Informer().AddEventHandler(handler)
	c.initInformer("deployment", c.SharedInformerFactory.Core().V1().Endpoints().Informer())
	return ch
}

func (c *client) ListServices(lo metav1.ListOptions, opt kconfig.Opt) ([]corev1.Service, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	items, err := inter.CoreV1().Services(opt.Namespace).List(lo)
	if err != nil {
		return nil, err
	}
	return items.Items, nil
}

func (c *client) GetService(name string, opt kconfig.Opt) (*corev1.Service, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.CoreV1().Services(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (c *client) UpdateService(service *corev1.Service, opt kconfig.Opt) (*corev1.Service, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.CoreV1().Services(opt.Namespace).Update(service)
}

func (c *client) WatchDeploy(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.AppsV1().Deployments(opt.Namespace).Watch(lo)
}

func (c *client) WaitTillDeployReady(name string, timeout time.Duration, opt kconfig.Opt) error {
	wi, err := c.WatchDeploy(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", name)}, opt)
	if err != nil {
		return err
	}

	timer := time.NewTimer(timeout)
	for {
		select {
		case <-timer.C:
			return except.NewError("Deploy failed to be ready after %s", except.ErrTimeout, timeout)
		case r := <-wi.ResultChan():
			switch r.Type {
			case watch.Error:
				reason := "unknown"
				if r.Object != nil {
					if dep, ok := r.Object.(*appsv1.Deployment); ok {
						if cond := c.getLatestCondition(dep); cond != nil {
							reason = cond.Message
						}
					}
				}
				return except.NewError("Deploy %s failed: %s", except.ErrUnknown, reason)
			case watch.Modified:
				if r.Object != nil {
					if dep, ok := r.Object.(*appsv1.Deployment); ok {
						if dep.Status.ReadyReplicas == dep.Status.Replicas {
							return nil
						}
					}
				}
			}
		}
	}
}

func (c *client) GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.CoreV1().Endpoints(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (c *client) DeleteDeploy(name string, opt kconfig.Opt) error {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return err
	}
	return inter.AppsV1().Deployments(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *client) GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.AppsV1().Deployments(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (c *client) UpsertDeploy(dep *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	deploy, err := inter.AppsV1().Deployments(opt.Namespace).Create(dep)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return inter.AppsV1().Deployments(opt.Namespace).Update(dep)
		}
	}
	return deploy, nil
}

func (c *client) ListConfigMaps(lo metav1.ListOptions, opt kconfig.Opt) ([]corev1.ConfigMap, error) {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}

	cmList, err := api.CoreV1().ConfigMaps(opt.Namespace).List(lo)
	if err != nil {
		return nil, err
	}

	cms := make([]corev1.ConfigMap, len(cmList.Items))
	for i, c := range cmList.Items {
		cms[i] = c
	}

	return cms, nil
}

func (c *client) DeleteConfigMap(name string, opt kconfig.Opt) error {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return err
	}

	return api.CoreV1().ConfigMaps(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *client) UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}

	cmApi := api.CoreV1().ConfigMaps(opt.Namespace)

	out, err := cmApi.Create(cm)
	if errors.IsAlreadyExists(err) {
		out, err = cmApi.Update(cm)
	}

	return out, err
}

func (c *client) GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}

	return api.CoreV1().ConfigMaps(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (c *client) WatchPods(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.CoreV1().Pods(opt.Namespace).Watch(lo)
}

func (c *client) getLatestCondition(dep *appsv1.Deployment) *appsv1.DeploymentCondition {
	if len(dep.Status.Conditions) > 0 {
		return &dep.Status.Conditions[len(dep.Status.Conditions)-1]
	}
	return nil
}

func (c *client) handlerFactory(filter Filter, caster func(obj interface{}) (runtime.Object, bool)) (chan watch.Event, cache.FilteringResourceEventHandler) {
	buffer := 100
	ch := make(chan watch.Event, buffer)
	filterHandler := cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			if v, ok := obj.(metav1.Object); ok {
				return filter(v)
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if v, ok := caster(obj); len(ch) < buffer && ok {
					ch <- watch.Event{
						Type:   watch.Added,
						Object: v,
					}
				}
			},
			UpdateFunc: func(_, newObj interface{}) {
				if v, ok := newObj.(runtime.Object); len(ch) < buffer && ok {
					ch <- watch.Event{
						Type:   watch.Modified,
						Object: v,
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if v, ok := obj.(runtime.Object); len(ch) < buffer && ok {
					ch <- watch.Event{
						Type:   watch.Deleted,
						Object: v,
					}
				}
			},
		},
	}

	return ch, filterHandler
}

func (c *client) initInformer(resource string, informer cache.SharedIndexInformer) {
	c.mapLock.Lock()
	defer c.mapLock.Unlock()
	if _, ok := c.informerHandlersByResource[resource]; !ok {
		ch := make(chan struct{})
		c.informerHandlersByResource[resource] = ch
		informer.Run(ch)
		return
	}
}
