package envoyutil

import (
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpointv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
)

func EndpointMatchesAddr(addr string, endpoint *endpointv2.Endpoint) bool {
	if v, ok := endpoint.Address.Address.(*corev2.Address_SocketAddress); ok {
		if v.SocketAddress.Address == addr {
			return true
		}
	}
	return false
}

func FindEndpointAddr(addr string, endpoints []endpointv2.Endpoint) (*endpointv2.Endpoint, int) {
	for i, v := range endpoints {
		if EndpointMatchesAddr(addr, &v) {
			return &v, i
		}
	}
	return nil, -1
}

func ContainsEndpointAddr(addr string, endpoints []endpointv2.Endpoint) bool {
	v, _ := FindEndpointAddr(addr, endpoints)
	return v != nil
}

func RemoveEndpointAddr(addr string, endpoints []endpointv2.Endpoint) []endpointv2.Endpoint {
	v, idx := FindEndpointAddr(addr, endpoints)
	if v != nil {
		return append(endpoints[:idx], endpoints[idx+1:]...)
	}
	return endpoints
}
