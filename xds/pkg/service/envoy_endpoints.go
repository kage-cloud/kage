package service

import (
	"fmt"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/ktypes/objconv"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/envoyutil"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const EnvoyEndpointsServiceKey = "EnvoyEndpointsService"

type EnvoyEndpointsService interface {
	StorePod(controllerType meta.ControllerType, xdsAnno *meta.Xds, pod *corev1.Pod) error
	RemovePod(pod *corev1.Pod) error
	Clean(nodeId string) error
}

type envoyEndpointsService struct {
	KubeReaderService KubeReaderService       `inject:"KubeReaderService"`
	LockdownService   ProxyService            `inject:"ProxyService"`
	EndpointFactory   factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory   factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
}

func (e *envoyEndpointsService) Clean(nodeId string) error {
	return e.StoreClient.Delete(nodeId)
}

func (e *envoyEndpointsService) RemovePod(pod *corev1.Pod) error {
	states := e.findStatesByAddress(pod.Status.PodIP)

	for _, state := range states {
		if state.Endpoints == nil {
			state.Endpoints = make([]*endpoint.ClusterLoadAssignment, 0)
		}
		if state.Listeners == nil {
			state.Listeners = make([]*listener.Listener, 0)
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
			log.WithField("node_id", state.NodeId).
				WithField("pod", pod.Name).
				WithField("namespace", pod.Namespace).
				Debug("Removing pod from control plane.")
			err := e.StoreClient.Set(&state)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *envoyEndpointsService) StorePod(controllerType meta.ControllerType, xdsAnno *meta.Xds, pod *corev1.Pod) error {
	state, err := e.StoreClient.Get(xdsAnno.Config.NodeId)
	opt := kconfig.Opt{Namespace: pod.Namespace}
	if err != nil {
		return err
	}
	if state.Endpoints == nil {
		state.Endpoints = make([]*endpoint.ClusterLoadAssignment, 0)
	}
	if state.Listeners == nil {
		state.Listeners = make([]*listener.Listener, 0)
	}

	changed := false
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			err, changed = e.updateState(state, cp.Protocol, cp.ContainerPort, pod, xdsAnno, controllerType)
			if err != nil {
				log.WithField("container", c.Name).
					WithField("pod", pod.Name).
					WithField("namespace", pod.Namespace).
					WithField("node_id", xdsAnno.Config.NodeId).
					WithField("protocol", cp.Protocol).
					WithField("port", cp.ContainerPort).
					WithField("ip", pod.Status.PodIP).
					WithError(err).
					Debug("Failed to add pod to Envoy config.")
				continue
			}
		}
	}

	svcsLi, err := e.KubeReaderService.ListSelected(pod.Labels, ktypes.KindService, opt)
	if err != nil {
		log.WithError(err).
			WithField("namespace", pod.Namespace).
			WithField("pod", pod.Name).
			Debug("Failed to get services for pod.")
	} else {
		for _, s := range svcsLi.(*corev1.ServiceList).Items {
			for _, port := range s.Spec.Ports {
				err, changed = e.updateState(state, port.Protocol, port.TargetPort.IntVal, pod, xdsAnno, controllerType)
				if err != nil {
					log.WithField("service", s.Name).
						WithField("pod", pod.Name).
						WithField("namespace", s.Namespace).
						WithField("node_id", xdsAnno.Config.NodeId).
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
		log.WithField("node_id", xdsAnno.Config.NodeId).
			WithField("pod", pod.Name).
			WithField("namespace", pod.Namespace).
			Debug("Adding pod to control plane.")
		err = e.StoreClient.Set(state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *envoyEndpointsService) updateState(state *store.EnvoyState, protocol corev1.Protocol, port int32, pod *corev1.Pod, xdsMeta *meta.Xds, controllerType meta.ControllerType) (error, bool) {
	proto, err := util.KubeProtocolToSocketAddressProtocol(protocol)
	changed := false
	podIp := pod.Status.PodIP
	if err != nil {
		log.WithField("port", port).
			WithField("ip", podIp).
			WithField("node_id", state.NodeId).
			WithField("protocol", proto).
			Debug("Protocol is not supported")
	}
	if !envoyutil.ContainsListenerPort(uint32(port), state.Listeners) {
		list, err := e.ListenerFactory.Listener(fmt.Sprintf("listener-%d", port), uint32(port), proto)
		if err != nil {
			return err, false
		}
		state.Listeners = append(state.Listeners, list)
		changed = true
	}

	envoyConf := xdsMeta.Config.Source
	if controllerType == meta.CanaryControllerType {
		envoyConf = xdsMeta.Config.Canary
	}

	if !envoyutil.ContainsEndpointAddr(podIp, state.Endpoints) {
		ep := e.EndpointFactory.Endpoint(envoyConf.ClusterName, proto, pod.Status.PodIP, uint32(port))
		logrus.WithField("cluster", envoyConf.ClusterName).WithField("pod", pod.Name).Debug("Adding endpoint from pod.")
		state.Endpoints = append(state.Endpoints, ep)
		changed = true
	}

	return nil, changed
}

func (e *envoyEndpointsService) getServicesForLabels(set labels.Set, namespace string) ([]corev1.Service, error) {
	svcs, err := e.KubeReaderService.ListServices(labels.Everything(), kconfig.Opt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	objs := kfilter.FilterObject(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			selector, err := e.LockdownService.GetSelector(v)
			if err == nil && !selector.Empty() {
				return selector.Matches(set)
			}
		}
		return false
	}, objconv.FromServices(svcs)...)

	svcs = make([]corev1.Service, len(objs))
	for i, v := range objs {
		svcs[i] = *v.(*corev1.Service)
	}
	return svcs, nil
}

func (e *envoyEndpointsService) findStatesByAddress(addr string) []store.EnvoyState {
	states := e.StoreClient.List()
	output := make([]store.EnvoyState, 0, len(states))
	for _, state := range states {
		eps := envoyutil.AggAllEndpoints(state.Endpoints)
		ep, _ := envoyutil.FindEndpointAddr(addr, eps)
		if ep != nil {
			output = append(output, state)
		}
	}
	return output
}
