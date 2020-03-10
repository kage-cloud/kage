package watcher

import (
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/snap/store"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const XdsWatcherKey = "XdsWatcher"

type XdsWatcher interface {
	WatchEndpoints(endpointNames []string, handlers ...model.EventHandler) error
}

type xdsWatcher struct {
	KubeClient      kube.Client             `inject:"KubeClient"`
	EndpointFactory factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
}

func (x *xdsWatcher) WatchEndpoints(endpointNames []string, handlers ...model.EventHandler) error {
	wi := x.KubeClient.InformEndpoints(func(object v1.Object) bool {
		for _, s := range endpointNames {
			if s == object.GetName() {
				return true
			}
		}
		return false
	})

	go func() {
		for e := range wi {
			endpoints := e.Object.(*corev1.Endpoints)
			ep := x.EndpointFactory.FromEndpoints(endpoints)
			list := x.ListenerFactory.FromEndpoints(endpoints)
			err := x.StoreClient.Set(&store.EnvoyState{
				Name:      endpoints.Name,
				Listeners: list,
				Endpoints: ep,
			})
			if err != nil {
				fmt.Println("Failed to set from endpoint", endpoints.Name, ":", err.Error())
			}
			for _, handler := range handlers {
				handler(e)
			}
		}
	}()
	return nil
}

func xdsWatcherFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&xdsWatcher{})
}

func (x *xdsWatcher) genWatchersKey(name, namespace, resource string) string {
	return fmt.Sprintf("%s-%s-%s", name, namespace, resource)
}
