package factory

import (
	"fmt"
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	listener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/kage-cloud/kage/xds/model"
)

const ListenerFactoryKey = "ListenerFactory"

type ListenerFactory interface {
	Listener(name string, port uint32, protocol envcore.SocketAddress_Protocol) (*apiv2.Listener, error)
}

func NewListenerFactory() ListenerFactory {
	return new(listenerFactory)
}

type listenerFactory struct {
}

func (l *listenerFactory) Listener(name string, port uint32, protocol envcore.SocketAddress_Protocol) (*apiv2.Listener, error) {
	manager := &hcm.HttpConnectionManager{
		StatPrefix: name,
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
				RouteConfigName: name,
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
		return nil, err
	}

	return &apiv2.Listener{
		Name: fmt.Sprintf("%s-%d", name, port),
		Address: &envcore.Address{
			Address: &envcore.Address_SocketAddress{
				SocketAddress: &envcore.SocketAddress{
					Protocol:      protocol,
					Address:       "0.0.0.0",
					PortSpecifier: &envcore.SocketAddress_PortValue{PortValue: port},
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
	}, nil
}
