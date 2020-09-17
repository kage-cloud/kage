package factory

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

const EndpointFactoryKey = "EndpointFactory"

type EndpointFactory interface {
	Endpoint(protocol core.SocketAddress_Protocol, address string, port uint32) *endpoint.Endpoint
}

func NewEndpointFactory() EndpointFactory {
	return new(endpointFactory)
}

type endpointFactory struct {
}

func (e *endpointFactory) Endpoint(protocol core.SocketAddress_Protocol, address string, port uint32) *endpoint.Endpoint {
	return &endpoint.Endpoint{
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol:      protocol,
					Address:       address,
					PortSpecifier: &core.SocketAddress_PortValue{PortValue: port},
				},
			},
		},
	}
}
