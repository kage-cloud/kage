package service

import (
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/envoyutil"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const XdsEventHandlerKey = "XdsEventHandler"

type XdsEventHandler interface {
	DeployPodsEventHandler(nodeId string) model.InformEventHandler
}

type xdsEventHandler struct {
	EndpointFactory factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
}

func (x *xdsEventHandler) DeployPodsEventHandler(nodeId string) model.InformEventHandler {
	return &model.InformEventHandlerFuncs{
		OnWatch: x.onWatch(nodeId),
		OnList:  x.onList(nodeId),
	}
}

func (x *xdsEventHandler) onList(nodeId string) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {
			for _, p := range v.Items {
				if err := x.storePod(nodeId, &p); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (x *xdsEventHandler) onWatch(nodeId string) model.OnWatchEventFunc {
	return func(event watch.Event) bool {
		if pod, ok := event.Object.(*corev1.Pod); ok {
			switch event.Type {
			case watch.Error, watch.Deleted:
				if err := x.removePod(nodeId, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", nodeId).
						WithError(err).
						Error("Failed to remove pod from control plane.")
				}
				break
			case watch.Modified, watch.Added:
				if err := x.storePod(nodeId, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", nodeId).
						WithError(err).
						Error("Failed to add pod to control plane.")
				}
			}
		}
		return true
	}
}

func (x *xdsEventHandler) removePod(nodeId string, pod *corev1.Pod) error {
	state, err := x.StoreClient.Get(nodeId)
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

func (x *xdsEventHandler) storePod(nodeId string, pod *corev1.Pod) error {
	state, err := x.StoreClient.Get(nodeId)
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
				log.WithField("container", c.Name).
					WithField("pod", pod.Name).
					WithField("namespace", pod.Namespace).
					WithField("protocol", cp.Protocol).
					Debug("Protocol is not supported")
				continue
			}
			if !envoyutil.ContainsListenerPort(uint32(cp.ContainerPort), state.Listeners) {
				listener, err := x.ListenerFactory.Listener(nodeId, uint32(cp.ContainerPort), proto)
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
	}
	return nil
}
