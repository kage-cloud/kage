package service

import (
	"fmt"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/model"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/snap/store"
	"github.com/kage-cloud/kage/xds/util"
	"github.com/kage-cloud/kage/xds/util/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

const XdsEventHandlerService = "XdsEventHandlerService"

type XdsEventHandler interface {
	DeployPodsEventHandler(deploy *appsv1.Deployment) model.InformEventHandler
}

type xdsWatcher struct {
	KubeClient      kube.Client             `inject:"KubeClient"`
	EndpointFactory factory.EndpointFactory `inject:"EndpointFactory"`
	ListenerFactory factory.ListenerFactory `inject:"ListenerFactory"`
	StoreClient     snap.StoreClient        `inject:"StoreClient"`
}

func (x *xdsWatcher) DeployPodsEventHandler(deploy *appsv1.Deployment) model.InformEventHandler {
	return &model.InformEventHandlerFuncs{
		OnWatch: x.onWatch(deploy),
		OnList:  x.onList(deploy),
	}
}

func (x *xdsWatcher) onList(deploy *appsv1.Deployment) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {
			for _, p := range v.Items {
				if err := x.storePod(kubeutil.ObjectKey(deploy), &p); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func (x *xdsWatcher) onWatch(deploy *appsv1.Deployment) model.OnWatchEventFunc {
	return func(event watch.Event) {
		if pod, ok := event.Object.(*corev1.Pod); ok {
			if err := x.storePod(kubeutil.ObjectKey(deploy), pod); err != nil {
				fmt.Println("Failed to set from pod", pod.Name, ":", err.Error())
			}
		}
	}
}

func (x *xdsWatcher) storePod(key string, pod *corev1.Pod) error {
	endpoints := make([]envoy_api_v2_endpoint.Endpoint, 0)
	listeners := make([]api.Listener, 0)
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			proto, err := util.KubeProtocolToSocketAddressProtocol(cp.Protocol)
			if err != nil {
				fmt.Println("Skipping container", c.Name, "for pod", pod.Name, "in namespace", pod.Namespace, "as its protocol is not supported")
			}
			listener, err := x.ListenerFactory.Listener(key, uint32(cp.ContainerPort), proto)
			if err != nil {
				return err
			}
			ep := x.EndpointFactory.Endpoint(proto, pod.Status.PodIP, uint32(cp.ContainerPort))
			endpoints = append(endpoints, *ep)
			listeners = append(listeners, *listener)
		}
	}
	err := x.StoreClient.Set(&store.EnvoyState{
		NodeId:    key,
		Listeners: listeners,
		Endpoints: endpoints,
	})
	if err != nil {
		return err
	}
	return nil
}
