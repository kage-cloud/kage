package kube

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube/kengine"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kinformer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sync"
)

type InformerClient interface {
	Informing(kind ktypes.NamespaceKind) bool

	Inform(ctx context.Context, spec kinformer.InformerSpec) error

	// Returns a Kind list so if the kind is a Deploy, an *appsv1.DeploymentList is returned.
	List(nsKind ktypes.NamespaceKind, selector labels.Selector) (runtime.Object, error)

	Get(nsKind ktypes.NamespaceKind, name string) (metav1.Object, error)
}

func NewInformerClient(apiClient Client) InformerClient {
	return &informerClient{
		Client:               apiClient,
		factoriesLock:        sync.RWMutex{},
		factoriesByNamespace: map[string]informers.SharedInformerFactory{},
		informedNsKinds:      map[ktypes.NamespaceKind]struct{}{},
	}
}

type informerClient struct {
	Client        Client
	factoriesLock sync.RWMutex

	factoriesByNamespace map[string]informers.SharedInformerFactory
	informedNsKinds      map[ktypes.NamespaceKind]struct{}
}

func (i *informerClient) Get(nsKind ktypes.NamespaceKind, name string) (metav1.Object, error) {
	fact := i.getFactory(nsKind)
	if fact == nil {
		return nil, except.NewError("No informer for %s is currently running", except.ErrNotFound, nsKind)
	}

	switch nsKind.Kind {

	case ktypes.KindDeploy:
		return fact.Apps().V1().Deployments().Lister().Deployments(nsKind.Namespace).Get(name)
	case ktypes.KindReplicaSet:
		return fact.Apps().V1().ReplicaSets().Lister().ReplicaSets(nsKind.Namespace).Get(name)
	case ktypes.KindService:
		return fact.Core().V1().Services().Lister().Services(nsKind.Namespace).Get(name)
	case ktypes.KindPod:
		return fact.Core().V1().Pods().Lister().Pods(nsKind.Namespace).Get(name)
	case ktypes.KindConfigMap:
		return fact.Core().V1().ConfigMaps().Lister().ConfigMaps(nsKind.Namespace).Get(name)
	case ktypes.KindEndpoints:
		return fact.Core().V1().Endpoints().Lister().Endpoints(nsKind.Namespace).Get(name)
	}

	return nil, except.NewError("Unsupported kind %s", except.ErrUnsupported, nsKind.Kind)
}

func (i *informerClient) List(nsKind ktypes.NamespaceKind, selector labels.Selector) (runtime.Object, error) {
	fact := i.getFactory(nsKind)
	if fact == nil {
		return nil, except.NewError("No informer for %s is currently running", except.ErrNotFound, nsKind)
	}

	switch nsKind.Kind {
	case ktypes.KindDeploy:
		list, err := fact.Apps().V1().Deployments().Lister().Deployments(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]appsv1.Deployment, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &appsv1.DeploymentList{Items: items}, nil

	case ktypes.KindReplicaSet:
		list, err := fact.Apps().V1().ReplicaSets().Lister().ReplicaSets(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]appsv1.ReplicaSet, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &appsv1.ReplicaSetList{Items: items}, nil

	case ktypes.KindConfigMap:
		list, err := fact.Core().V1().ConfigMaps().Lister().ConfigMaps(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]corev1.ConfigMap, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &corev1.ConfigMapList{Items: items}, nil

	case ktypes.KindPod:
		list, err := fact.Core().V1().Pods().Lister().Pods(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]corev1.Pod, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &corev1.PodList{Items: items}, nil

	case ktypes.KindService:
		list, err := fact.Core().V1().Services().Lister().Services(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]corev1.Service, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &corev1.ServiceList{Items: items}, nil

	case ktypes.KindEndpoints:
		list, err := fact.Core().V1().Endpoints().Lister().Endpoints(nsKind.Namespace).List(selector)
		if err != nil {
			return nil, err
		}

		items := make([]corev1.Endpoints, len(list))
		for i, v := range list {
			items[i] = *v
		}

		return &corev1.EndpointsList{Items: items}, nil
	}

	return nil, except.NewError("Unsupported kind %s", except.ErrUnsupported, nsKind.Kind)
}

func (i *informerClient) Informing(nsKind ktypes.NamespaceKind) bool {
	i.factoriesLock.RLock()
	defer i.factoriesLock.RUnlock()
	_, ok := i.informedNsKinds[nsKind]
	return ok
}

func (i *informerClient) Inform(ctx context.Context, spec kinformer.InformerSpec) error {
	fact := i.lazyGetFactory(spec.NamespaceKind)

	informer, err := i.informerForKind(spec.NamespaceKind.Kind, fact)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	queue := i.createHandlerQueue(ctx, spec)

	informer.AddEventHandler(i.handlerFactory(queue, spec))

	i.runInformer(informer)

	i.waitForSync(informer)

	obj, err := i.List(spec.NamespaceKind, labels.Everything())
	if err != nil {
		return err
	}
	if spec.Filter != nil {
		li := obj.(metav1.ListInterface)
		objs := kubeutil.ObjectsFromList(li)
		filteredObjs := make([]runtime.Object, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(metav1.Object); ok && spec.Filter(v) {
				filteredObjs = append(filteredObjs, o)
			}
		}
		obj = kubeutil.ToListType(spec.NamespaceKind.Kind, filteredObjs)
	}

	for _, h := range spec.Handlers {
		if err := h.OnListEvent(obj); err != nil {
			cancel()
			return err
		}
	}

	return nil
}

func (i *informerClient) lazyGetFactory(nsKind ktypes.NamespaceKind) informers.SharedInformerFactory {
	i.factoriesLock.Lock()
	defer i.factoriesLock.Unlock()
	fact, ok := i.factoriesByNamespace[nsKind.Namespace]
	if !ok {
		fact = informers.NewSharedInformerFactoryWithOptions(i.Client.Api(), 0, informers.WithNamespace(nsKind.Namespace))
		i.factoriesByNamespace[nsKind.Namespace] = fact
	}
	i.informedNsKinds[nsKind] = struct{}{}
	return fact
}

func (i *informerClient) getFactory(nsKind ktypes.NamespaceKind) informers.SharedInformerFactory {
	i.factoriesLock.RLock()
	defer i.factoriesLock.RUnlock()
	return i.factoriesByNamespace[nsKind.Namespace]
}

func (i *informerClient) createHandlerQueue(ctx context.Context, spec kinformer.InformerSpec) kengine.HandlerQueue {
	queue := kengine.NewHandlerQueue(spec.Handlers...)
	queue.Start(ctx)
	return queue
}

func (i *informerClient) informerForKind(kind ktypes.Kind, factory informers.SharedInformerFactory) (inf cache.SharedIndexInformer, err error) {
	switch kind {
	case ktypes.KindConfigMap:
		inf = factory.Core().V1().ConfigMaps().Informer()
	case ktypes.KindDeploy:
		inf = factory.Apps().V1().Deployments().Informer()
	case ktypes.KindPod:
		inf = factory.Core().V1().Pods().Informer()
	case ktypes.KindReplicaSet:
		inf = factory.Apps().V1().ReplicaSets().Informer()
	case ktypes.KindService:
		inf = factory.Core().V1().Services().Informer()
	case ktypes.KindEndpoints:
		inf = factory.Core().V1().Endpoints().Informer()
	}
	if inf == nil {
		err = except.NewError("Kind %s not supported", except.ErrUnsupported, kind)
	}
	return
}

func (i *informerClient) handlerFactory(queue workqueue.DelayingInterface, spec kinformer.InformerSpec) cache.FilteringResourceEventHandler {
	return cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			if v, ok := obj.(metav1.Object); ok {
				return spec.Filter(v)
			}
			return false
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if v, ok := obj.(runtime.Object); ok {
					queue.AddAfter(watch.Event{
						Type:   watch.Added,
						Object: v,
					}, spec.BatchDuration)
				}
			},
			UpdateFunc: func(_, obj interface{}) {
				if v, ok := obj.(runtime.Object); ok {
					queue.AddAfter(watch.Event{
						Type:   watch.Modified,
						Object: v,
					}, spec.BatchDuration)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if v, ok := obj.(runtime.Object); ok {
					queue.AddAfter(watch.Event{
						Type:   watch.Deleted,
						Object: v,
					}, spec.BatchDuration)
				}
			},
		},
	}
}

func (i *informerClient) waitForSync(informer cache.SharedIndexInformer) {
	cache.WaitForCacheSync(make(chan struct{}), func() bool {
		return informer.HasSynced()
	})
}

func (i *informerClient) runInformer(informer cache.SharedIndexInformer) {
	if !informer.HasSynced() {
		go func() {
			informer.Run(make(chan struct{}))
		}()
	}
}
