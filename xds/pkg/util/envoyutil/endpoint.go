package envoyutil

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
)

func EndpointMatchesAddr(addr string, endpoint *endpoint.Endpoint) bool {
	if v, ok := endpoint.Address.Address.(*core.Address_SocketAddress); ok {
		if v.SocketAddress.Address == addr {
			return true
		}
	}
	return false
}

func FindEndpointAddr(addr string, endpoints []endpoint.Endpoint) (*endpoint.Endpoint, int) {
	for i, v := range endpoints {
		if EndpointMatchesAddr(addr, &v) {
			return &v, i
		}
	}
	return nil, -1
}

func ContainsEndpointAddr(addr string, endpoints []endpoint.Endpoint) bool {
	v, _ := FindEndpointAddr(addr, endpoints)
	return v != nil
}

func RemoveEndpointAddr(addr string, endpoints []endpoint.Endpoint) []endpoint.Endpoint {
	v, idx := FindEndpointAddr(addr, endpoints)
	if v != nil {
		return append(endpoints[:idx], endpoints[idx+1:]...)
	}
	return endpoints
}
