package factory

import (
	"fmt"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/util"
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	corev1 "k8s.io/api/core/v1"
)

const ListenerFactoryKey = "ListenerFactory"

type ListenerFactory interface {
	FromEndpoints(endpoints *corev1.Endpoints) []apiv2.Listener
}

func NewListenerFactory() ListenerFactory {
	return new(listenerFactory)
}

type listenerFactory struct {
}

func (l *listenerFactory) FromEndpoints(endpoints *corev1.Endpoints) []apiv2.Listener {
	listeners := make([]apiv2.Listener, 0)
	for _, ss := range endpoints.Subsets {
		for _, port := range ss.Ports {
			proto, err := util.KubeProtocolToSocketAddressProtocol(port.Protocol)
			if err != nil {
				fmt.Println("Failed to map protocol for kube endpoint: ", port.Protocol)
				continue
			}

			manager := &hcm.HttpConnectionManager{
				StatPrefix: endpoints.Name,
				RouteSpecifier: &hcm.HttpConnectionManager_Rds{
					Rds: &hcm.Rds{
						ConfigSource: &envcore.ConfigSource{
							ConfigSourceSpecifier: &envcore.ConfigSource_ApiConfigSource{
								ApiConfigSource: &envcore.ApiConfigSource{
									ApiType: envcore.ApiConfigSource_GRPC,
									GrpcServices: []*envcore.GrpcService{
										{
											TargetSpecifier: &envcore.GrpcService_EnvoyGrpc_{
												EnvoyGrpc: &envcore.GrpcService_EnvoyGrpc{
													ClusterName: model.XdsClusterName,
												},
											},
										},
									},
									SetNodeOnFirstMessageOnly: true,
								},
							},
						},
						RouteConfigName: endpoints.Name,
					},
				},
				HttpFilters: []*hcm.HttpFilter{
					{
						Name: wellknown.Router,
					},
				},
			}

			hcmAny, err := ptypes.MarshalAny(manager)
			if err != nil {
				fmt.Println("Failed to marshal the connection manager for the", endpoints.Name, "endpoint:", err.Error())
				continue
			}

			lis := apiv2.Listener{
				Name: fmt.Sprintf("%s-%d", endpoints.Name, port.Port),
				Address: &envcore.Address{
					Address: &envcore.Address_SocketAddress{
						SocketAddress: &envcore.SocketAddress{
							Protocol:      proto,
							Address:       "0.0.0.0",
							PortSpecifier: &envcore.SocketAddress_PortValue{PortValue: uint32(port.Port)},
						},
					},
				},
				FilterChains: []*listener.FilterChain{
					{
						Filters: []*listener.Filter{
							{
								Name: wellknown.HTTPConnectionManager,
								ConfigType: &listener.Filter_TypedConfig{
									TypedConfig: hcmAny,
								},
							},
						},
					},
				},
			}

			listeners = append(listeners, lis)
		}
	}

	return listeners
}
