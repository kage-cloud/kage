package service

import (
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/ktypes/objconv"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kfilter"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kinformer"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/envoyutil"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const XdsEventHandlerKey = "XdsEventHandler"

type XdsEventHandler interface {
	DeployPodsEventHandler(nodeId string) kinformer.InformEventHandler
}

type xdsEventHandler struct {
	EndpointFactory   factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory   factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
	KubeReaderService KubeReaderService       `inject:"KubeReaderService"`
}

func (x *xdsEventHandler) DeployPodsEventHandler(nodeId string) kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: x.onWatch(nodeId),
		OnList:  x.onList(nodeId),
	}
}

func (x *xdsEventHandler) onList(nodeId string) kinformer.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {
			for _, p := range v.Items {
				if util.IsKageMesh(p.Labels) {
					continue
				}
				if err := x.storePod(nodeId, &p); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (x *xdsEventHandler) onWatch(nodeId string) kinformer.OnWatchEventFunc {
	return func(event watch.Event) error {
		if pod, ok := event.Object.(*corev1.Pod); ok {
			if util.IsKageMesh(pod.Labels) {
				return nil
			}
			switch event.Type {
			case watch.Error, watch.Deleted:
				if err := x.removePod(nodeId, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", nodeId).
						WithError(err).
						Error("Failed to remove pod from control plane.")
					return err
				}
				break
			case watch.Modified, watch.Added:
				if err := x.storePod(nodeId, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", nodeId).
						WithError(err).
						Error("Failed to add pod to control plane.")
					return err
				}
			}
		}
		return nil
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
		log.WithField("node_id", nodeId).
			WithField("pod", pod.Name).
			WithField("namespace", pod.Namespace).
			Debug("Removing pod from control plane.")
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
			err, changed = x.updateState(state, cp.Protocol, cp.ContainerPort, pod.Status.PodIP, nodeId)
			if err != nil {
				log.WithField("container", c.Name).
					WithField("pod", pod.Name).
					WithField("namespace", pod.Namespace).
					WithField("node_id", nodeId).
					WithField("protocol", cp.Protocol).
					WithField("port", cp.ContainerPort).
					WithField("ip", pod.Status.PodIP).
					WithError(err).
					Debug("Failed to add pod to Envoy config.")
				continue
			}
		}
	}

	svcs, err := x.getServicesForLabels(pod.Labels, pod.Namespace)
	if err != nil {
		log.WithError(err).
			WithField("namespace", pod.Namespace).
			WithField("pod", pod.Name).
			Debug("Failed to get services for pod.")
	} else {
		for _, s := range svcs {
			for _, port := range s.Spec.Ports {
				err, changed = x.updateState(state, port.Protocol, port.TargetPort.IntVal, pod.Status.PodIP, nodeId)
				if err != nil {
					log.WithField("service", s.Name).
						WithField("pod", pod.Name).
						WithField("namespace", s.Namespace).
						WithField("node_id", nodeId).
						WithField("protocol", port.Protocol).
						WithField("port", port.TargetPort.IntVal).
						WithField("ip", pod.Status.PodIP).
						WithError(err).
						Debug("Failed to add pod's service to Envoy config.")
					continue
				}
			}
		}
	}

	if changed {
		log.WithField("node_id", nodeId).
			WithField("pod", pod.Name).
			WithField("namespace", pod.Namespace).
			Debug("Adding pod to control plane.")
		err = x.StoreClient.Set(state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *xdsEventHandler) updateState(state *store.EnvoyState, protocol corev1.Protocol, port int32, ip string, nodeId string) (error, bool) {
	proto, err := util.KubeProtocolToSocketAddressProtocol(protocol)
	changed := false
	if err != nil {
		log.WithField("port", port).
			WithField("ip", ip).
			WithField("node_id", nodeId).
			WithField("protocol", proto).
			Debug("Protocol is not supported")
	}
	if !envoyutil.ContainsListenerPort(uint32(port), state.Listeners) {
		listener, err := x.ListenerFactory.Listener(nodeId, uint32(port), proto)
		if err != nil {
			return err, false
		}
		state.Listeners = append(state.Listeners, *listener)
		changed = true
	}
	if !envoyutil.ContainsEndpointAddr(ip, state.Endpoints) {
		ep := x.EndpointFactory.Endpoint(proto, ip, uint32(port))
		state.Endpoints = append(state.Endpoints, *ep)
		changed = true
	}

	return nil, changed
}

func (x *xdsEventHandler) getServicesForLabels(set labels.Set, namespace string) ([]corev1.Service, error) {
	svcs, err := x.KubeReaderService.ListServices(labels.Everything(), kconfig.Opt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	objs := kfilter.FilterObject(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			return labels.SelectorFromSet(v.Spec.Selector).Matches(set)
		}
		return false
	}, objconv.FromServices(svcs)...)

	svcs = make([]corev1.Service, len(objs))
	for i, v := range objs {
		svcs[i] = *v.(*corev1.Service)
	}
	return svcs, nil
}
