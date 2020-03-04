package watcher

import (
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/snap/store"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const XdsWatcherKey = "XdsWatcher"

type XdsWatcher interface {
	WatchEndpoints(name string, opt kconfig.Opt) error
	StopWatchingEndpoints(name string, opt kconfig.Opt)
}

type xdsWatcher struct {
	KubeClient      kube.Client             `inject:"KubeClient"`
	EndpointFactory factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
	watchers        map[string]watch.Interface
}

func (x *xdsWatcher) StopWatchingEndpoints(name string, opt kconfig.Opt) {
	if wi, ok := x.watchers[x.genWatchersKey(name, opt.Namespace, "endpoints")]; ok {
		wi.Stop()
	}
}

func (x *xdsWatcher) WatchEndpoints(name string, opt kconfig.Opt) error {
	lo := v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	}

	wi, err := x.KubeClient.WatchEndpoints(lo, opt)
	if err != nil {
		return err
	}

	if _, ok := x.watchers[x.genWatchersKey(name, opt.Namespace, "endpoints")]; ok {
		fmt.Println("Already watching endpoints", name, "in", opt.Namespace)
		return nil
	}

	go func() {
		for e := range wi.ResultChan() {
			endpoints := e.Object.(*corev1.Endpoints)
			ep := x.EndpointFactory.FromEndpoints(endpoints)
			list := x.ListenerFactory.FromEndpoints(endpoints)
			err := x.StoreClient.Set(&store.EnvoyState{
				Name:      name,
				Listeners: list,
				Endpoints: ep,
			})
			if err != nil {
				fmt.Println("Failed to set from endpoint", name, ":", err.Error())
			}
		}
	}()
	return nil
}

func xdsWatcherFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&xdsWatcher{
		watchers: make(map[string]watch.Interface),
	})
}

func (x *xdsWatcher) genWatchersKey(name, namespace, resource string) string {
	return fmt.Sprintf("%s-%s-%s", name, namespace, resource)
}
