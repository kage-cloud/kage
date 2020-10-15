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

const CanaryEndpointsServiceKey = "CanaryEndpointsService"

type CanaryEndpointsService interface {
	StorePod(pod *corev1.Pod) error
	RemovePod(pod *corev1.Pod) error
	Clean(nodeId string) error
}

type canaryEndpointsService struct {
	KubeReaderService KubeReaderService       `inject:"KubeReaderService"`
	LockdownService   ProxyService            `inject:"ProxyService"`
	EndpointFactory   factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory   factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
	KageMeshService   KageMeshService         `inject:"KageMeshService"`
	CanaryService     CanaryService           `inject:"CanaryService"`
}

func (c *canaryEndpointsService) StorePod(pod *corev1.Pod) error {
	meshes, err := c.KageMeshService.ListXdsForPod(pod)
	if err != nil {
		return err
	}

	canaryAnno := c.CanaryService.FetchForPod(pod)
	if canaryAnno == nil {
		for _, mesh := range meshes {
			if err := c.storePod(meta.SourceControllerType, &mesh, pod); err != nil {
				return err
			}
		}

		return nil
	}

	if len(meshes) == 0 {
		xdsAnno, err := c.KageMeshService.CreateForCanary(canaryAnno)
		if err != nil {
			return err
		}
		meshes = append(meshes, *xdsAnno)
	}

	for _, mesh := range meshes {
		if err := c.storePod(meta.CanaryControllerType, &mesh, pod); err != nil {
			return err
		}
	}

	return nil
}

func (c *canaryEndpointsService) Clean(nodeId string) error {
	return c.StoreClient.Delete(nodeId)
}

func (c *canaryEndpointsService) RemovePod(pod *corev1.Pod) error {
	states := c.findStatesByAddress(pod.Status.PodIP)

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
			err := c.StoreClient.Set(&state)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *canaryEndpointsService) storePod(controllerType meta.ControllerType, xdsAnno *meta.Xds, pod *corev1.Pod) error {
	state, err := c.StoreClient.Get(xdsAnno.Config.NodeId)
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
	for _, container := range pod.Spec.Containers {
		for _, cp := range container.Ports {
			err, changed = c.updateState(state, cp.Protocol, cp.ContainerPort, pod, xdsAnno, controllerType)
			if err != nil {
				log.WithField("container", container.Name).
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

	svcsLi, err := c.KubeReaderService.ListSelected(pod.Labels, ktypes.KindService, opt)
	if err != nil {
		log.WithError(err).
			WithField("namespace", pod.Namespace).
			WithField("pod", pod.Name).
			Debug("Failed to get services for pod.")
	} else {
		for _, s := range svcsLi.(*corev1.ServiceList).Items {
			for _, port := range s.Spec.Ports {
				err, changed = c.updateState(state, port.Protocol, port.TargetPort.IntVal, pod, xdsAnno, controllerType)
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
		err = c.StoreClient.Set(state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *canaryEndpointsService) updateState(state *store.EnvoyState, protocol corev1.Protocol, port int32, pod *corev1.Pod, xdsMeta *meta.Xds, controllerType meta.ControllerType) (error, bool) {
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
		list, err := c.ListenerFactory.Listener(fmt.Sprintf("listener-%d", port), uint32(port), proto)
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
		ep := c.EndpointFactory.Endpoint(envoyConf.ClusterName, proto, pod.Status.PodIP, uint32(port))
		logrus.WithField("cluster", envoyConf.ClusterName).WithField("pod", pod.Name).Debug("Adding endpoint from pod.")
		state.Endpoints = append(state.Endpoints, ep)
		changed = true
	}

	return nil, changed
}

func (c *canaryEndpointsService) getServicesForLabels(set labels.Set, namespace string) ([]corev1.Service, error) {
	svcs, err := c.KubeReaderService.ListServices(labels.Everything(), kconfig.Opt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	objs := kfilter.FilterObject(func(object metav1.Object) bool {
		if v, ok := object.(*corev1.Service); ok {
			selector, err := c.LockdownService.GetSelector(v)
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

func (c *canaryEndpointsService) findStatesByAddress(addr string) []store.EnvoyState {
	states := c.StoreClient.List()
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
