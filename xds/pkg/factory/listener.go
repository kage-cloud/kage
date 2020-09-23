package factory

import (
	"fmt"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envcore "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	fileaccessloggers "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/golang/protobuf/ptypes"
	"github.com/kage-cloud/kage/xds/pkg/model"
)

const ListenerFactoryKey = "ListenerFactory"

type ListenerFactory interface {
	Listener(name string, port uint32, protocol envcore.SocketAddress_Protocol) (*listener.Listener, error)
}

func NewListenerFactory() ListenerFactory {
	return new(listenerFactory)
}

type listenerFactory struct {
}

func (l *listenerFactory) Listener(name string, port uint32, protocol envcore.SocketAddress_Protocol) (*listener.Listener, error) {
	accesslogFilter := &fileaccessloggers.FileAccessLog{
		Path: "/dev/stdout",
	}
	alfAny, err := ptypes.MarshalAny(accesslogFilter)
	if err != nil {
		return nil, err
	}
	manager := &hcm.HttpConnectionManager{
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: "nginx-nginx-kage-canary",
				ConfigSource: &envcore.ConfigSource{
					ResourceApiVersion: envcore.ApiVersion_V3,
					ConfigSourceSpecifier: &envcore.ConfigSource_ApiConfigSource{
						ApiConfigSource: &envcore.ApiConfigSource{
							ApiType:             envcore.ApiConfigSource_GRPC,
							TransportApiVersion: envcore.ApiVersion_V3,
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

	return &listener.Listener{
		Name: fmt.Sprintf("%s-%d", name, port),
		AccessLog: []*accesslog.AccessLog{
			{
				Name: wellknown.FileAccessLog,
				ConfigType: &accesslog.AccessLog_TypedConfig{
					TypedConfig: alfAny,
				},
			},
		},
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
