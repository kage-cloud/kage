package factory

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

const EndpointFactoryKey = "EndpointFactory"

type EndpointFactory interface {
	Endpoint(clusterName string, protocol core.SocketAddress_Protocol, address string, port uint32) *endpoint.ClusterLoadAssignment
}

func NewEndpointFactory() EndpointFactory {
	return new(endpointFactory)
}

type endpointFactory struct {
}

func (e *endpointFactory) Endpoint(clusterName string, protocol core.SocketAddress_Protocol, address string, port uint32) *endpoint.ClusterLoadAssignment {
	return &endpoint.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []*endpoint.LocalityLbEndpoints{
			{
				LbEndpoints: []*endpoint.LbEndpoint{
					{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Protocol:      protocol,
											Address:       address,
											PortSpecifier: &core.SocketAddress_PortValue{PortValue: port},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
