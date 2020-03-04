package factory

import (
	"fmt"
	"github.com/eddieowens/kage/xds/util"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	corev1 "k8s.io/api/core/v1"
)

const EndpointFactoryKey = "EndpointFactory"

type EndpointFactory interface {
	FromEndpoints(endpoints *corev1.Endpoints) []endpoint.Endpoint
}

func NewEndpointFactory() EndpointFactory {
	return new(endpointFactory)
}

type endpointFactory struct {
}

func (e *endpointFactory) FromEndpoints(endpoints *corev1.Endpoints) []endpoint.Endpoint {
	eps := make([]endpoint.Endpoint, 0)
	for _, ss := range endpoints.Subsets {
		for _, add := range ss.Addresses {
			for _, port := range ss.Ports {
				proto, err := util.KubeProtocolToSocketAddressProtocol(port.Protocol)
				if err != nil {
					fmt.Println("Failed to map protocol for kube endpoint: ", port.Protocol)
					continue
				}
				ep := endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol:      proto,
								Address:       add.IP,
								PortSpecifier: &core.SocketAddress_PortValue{PortValue: uint32(port.Port)},
							},
						},
					},
				}
				eps = append(eps, ep)
			}
		}
	}
	return eps
}
