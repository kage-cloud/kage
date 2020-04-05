package service

import (
	"fmt"
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/model"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/util"
	"github.com/kage-cloud/kage/xds/util/envoyutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const XdsEventHandlerServiceKey = "XdsEventHandlerService"

type XdsEventHandler interface {
	DeployPodsEventHandler(deploy ...*appsv1.Deployment) model.InformEventHandler
}

type xdsEventHandler struct {
	KubeClient      kube.Client             `inject:"KubeClient"`
	EndpointFactory factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
}

func (x *xdsEventHandler) DeployPodsEventHandler(deploy ...*appsv1.Deployment) model.InformEventHandler {
	return &model.InformEventHandlerFuncs{
		OnWatch: x.onWatch(deploy...),
		OnList:  x.onList(deploy...),
	}
}

func (x *xdsEventHandler) onList(deploy ...*appsv1.Deployment) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {
			for _, d := range deploy {
				for _, p := range v.Items {
					if err := x.storePod(kubeutil.ObjectKey(d), &p); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func (x *xdsEventHandler) onWatch(deploy ...*appsv1.Deployment) model.OnWatchEventFunc {
	return func(event watch.Event) {
		if pod, ok := event.Object.(*corev1.Pod); ok {
			switch event.Type {
			case watch.Error, watch.Deleted:
				for _, d := range deploy {
					if err := x.removePod(kubeutil.ObjectKey(d), pod); err != nil {
						fmt.Println("Failed to set from pod", pod.Name, ":", err.Error())
					}
				}
				break
			case watch.Modified, watch.Added:
				for _, d := range deploy {
					if err := x.storePod(kubeutil.ObjectKey(d), pod); err != nil {
						fmt.Println("Failed to set from pod", pod.Name, ":", err.Error())
					}
				}
			}
		}
	}
}

func (x *xdsEventHandler) removePod(key string, pod *corev1.Pod) error {
	state, err := x.StoreClient.Get(key)
	if err != nil {
		return err
	}

	if state.Endpoints == nil {
		state.Endpoints = make([]endpointv2.Endpoint, 0)
	}
	if state.Listeners == nil {
		state.Listeners = make([]apiv2.Listener, 0)
	}

	changed := false
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			if envoyutil.ContainsListenerPort(uint32(cp.ContainerPort), state.Listeners) {
				changed = true
				state.Listeners = envoyutil.RemoveListenerPort(uint32(cp.ContainerPort), state.Listeners)
			}
			if envoyutil.ContainsEndpointAddr(pod.Status.PodIP, state.Endpoints) {
				changed = true
				state.Endpoints = envoyutil.RemoveEndpointAddr(pod.Status.PodIP, state.Endpoints)
			}
		}
	}

	if changed {
		err = x.StoreClient.Set(state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *xdsEventHandler) storePod(key string, pod *corev1.Pod) error {
	state, err := x.StoreClient.Get(key)
	if err != nil {
		return err
	}
	if state.Endpoints == nil {
		state.Endpoints = make([]endpointv2.Endpoint, 0)
	}
	if state.Listeners == nil {
		state.Listeners = make([]apiv2.Listener, 0)
	}

	changed := false
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			proto, err := util.KubeProtocolToSocketAddressProtocol(cp.Protocol)
			if err != nil {
				fmt.Println("Skipping container", c.Name, "for pod", pod.Name, "in namespace", pod.Namespace, "as its protocol is not supported")
				continue
			}
			if !envoyutil.ContainsListenerPort(uint32(cp.ContainerPort), state.Listeners) {
				listener, err := x.ListenerFactory.Listener(key, uint32(cp.ContainerPort), proto)
				if err != nil {
					return err
				}
				state.Listeners = append(state.Listeners, *listener)
				changed = true
			}
			if !envoyutil.ContainsEndpointAddr(pod.Status.PodIP, state.Endpoints) {
				ep := x.EndpointFactory.Endpoint(proto, pod.Status.PodIP, uint32(cp.ContainerPort))
				state.Endpoints = append(state.Endpoints, *ep)
				changed = true
			}
		}
	}

	if changed {
		err = x.StoreClient.Set(state)
		if err != nil {
			return err
		}
		return nil
	}
}
