package service

import (
	"context"
	"fmt"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/ktypes/objconv"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/envoyepctlr"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/envoyutil"
	"github.com/opencontainers/runc/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const EnvoyEndpointControllerKey = "EnvoyEndpointController"

type EnvoyEndpointController interface {
	StartAsync(ctx context.Context, spec *envoyepctlr.Spec) error
}

type envoyEndpointController struct {
	EndpointFactory   factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory   factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
	KubeReaderService KubeReaderService       `inject:"KubeReaderService"`
	WatchService      WatchService            `inject:"WatchService"`
	LockdownService   LockdownService         `inject:"LockdownService"`
}

func (e *envoyEndpointController) StartAsync(ctx context.Context, spec *envoyepctlr.Spec) error {
	selectors := make([]labels.Selector, 0, len(spec.PodClusters))
	for _, v := range spec.PodClusters {
		selectors = append(selectors, v.Selector)
	}

	return e.WatchService.Services(ctx, func(object metav1.Object) bool {
		return e.LockdownService.IsLockedDown(object)
	}, 3*time.Second, spec.Opt)
}

func (e *envoyEndpointController) name(opt kconfig.Opt) kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: nil,
		OnList: func(obj metav1.ListInterface) error {
			svcs, ok := obj.(*corev1.ServiceList)
			if !ok {
				return except.NewError("expected a service list but got %T", except.ErrInternalError, obj)
			}

			for _, v := range svcs.Items {
				selector, err := e.LockdownService.GetSelector(&v)
				if err != nil || selector.Empty() {
					log.WithField("selector", selector).
						WithError(err).
						WithField("name", v.Name).
						WithField("namespace", v.Namespace).
						Debug("Could not add envoy endpoints for service as no selector could be found.")
					continue
				}

				pods, err := e.KubeReaderService.ListPods(selector, opt)
				if err != nil {
					log.WithField("selector", selector).
						WithError(err).
						WithField("name", v.Name).
						WithField("namespace", v.Namespace).
						Debug("Could not add envoy endpoints for service. Failed to list pods for service's selector.")
					continue
				}

				for _, v := range pods {

				}
			}
		},
	}
}

func (e *envoyEndpointController) DeployPodsEventHandler(spec *envoyepctlr.Spec) kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: e.onWatch(spec),
		OnList:  e.onList(spec),
	}
}

func (e *envoyEndpointController) onList(spec *envoyepctlr.Spec) kinformer.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {
			for _, p := range v.Items {
				if util.IsKageMesh(p.Labels) {
					continue
				}
				if err := e.storePod(spec, &p); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (e *envoyEndpointController) onWatch(spec *envoyepctlr.Spec) kinformer.OnWatchEventFunc {
	return func(event watch.Event) error {
		if pod, ok := event.Object.(*corev1.Pod); ok {
			if util.IsKageMesh(pod.Labels) {
				return nil
			}
			switch event.Type {
			case watch.Error, watch.Deleted:
				if err := e.removePod(spec.NodeId, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", spec.NodeId).
						WithError(err).
						Error("Failed to remove pod from control plane.")
					return err
				}
				break
			case watch.Modified, watch.Added:
				if err := e.storePod(spec, pod); err != nil {
					log.WithField("name", pod.Name).
						WithField("namespace", pod.Namespace).
						WithField("node_id", spec.NodeId).
						WithError(err).
						Error("Failed to add pod to control plane.")
					return err
				}
			}
		}
		return nil
	}
}

func (e *envoyEndpointController) removePod(nodeId string, pod *corev1.Pod) error {
	state, err := e.StoreClient.Get(nodeId)
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
		err = e.StoreClient.Set(state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *envoyEndpointController) storePod(meta *model.XdsMeta, pod *corev1.Pod) error {
	state, err := e.StoreClient.Get(meta.NodeId)
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
			err, changed = e.updateState(state, cp.Protocol, cp.ContainerPort, pod, meta)
			if err != nil {
				log.WithField("container", c.Name).
					WithField("pod", pod.Name).
					WithField("namespace", pod.Namespace).
					WithField("node_id", meta.NodeId).
					WithField("protocol", cp.Protocol).
					WithField("port", cp.ContainerPort).
					WithField("ip", pod.Status.PodIP).
					WithError(err).
					Debug("Failed to add pod to Envoy config.")
				continue
			}
		}
	}

	svcs, err := e.getServicesForLabels(pod.Labels, pod.Namespace)
	if err != nil {
		log.WithError(err).
			WithField("namespace", pod.Namespace).
			WithField("pod", pod.Name).
			Debug("Failed to get services for pod.")
	} else {
		for _, s := range svcs {
			for _, port := range s.Spec.Ports {
				err, changed = e.updateState(state, port.Protocol, port.TargetPort.IntVal, pod, meta)
				if err != nil {
					log.WithField("service", s.Name).
						WithField("pod", pod.Name).
						WithField("namespace", s.Namespace).
						WithField("node_id", meta.NodeId).
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
		log.WithField("node_id", meta.NodeId).
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

func (e *envoyEndpointController) updateState(state *store.EnvoyState, protocol corev1.Protocol, port int32, pod *corev1.Pod, meta *model.XdsMeta) (error, bool) {
	proto, err := util.KubeProtocolToSocketAddressProtocol(protocol)
	changed := false
	podIp := pod.Status.PodIP
	if err != nil {
		log.WithField("port", port).
			WithField("ip", podIp).
			WithField("node_id", meta.NodeId).
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

	if !envoyutil.ContainsEndpointAddr(podIp, state.Endpoints) {
		ep := e.EndpointFactory.Endpoint(meta.ClusterName, proto, pod.Status.PodIP, uint32(port))
		logrus.WithField("cluster", meta.ClusterName).WithField("pod", pod.Name).Debug("Adding endpoint from pod.")
		state.Endpoints = append(state.Endpoints, ep)
		changed = true
	}

	return nil, changed
}

func (e *envoyEndpointController) getServicesForLabels(set labels.Set, namespace string) ([]corev1.Service, error) {
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
