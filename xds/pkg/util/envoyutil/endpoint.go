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

func FindEndpointAddr(addr string, endpoints []*endpoint.Endpoint) (*endpoint.Endpoint, int) {
	for i := range endpoints {
		if EndpointMatchesAddr(addr, endpoints[i]) {
			return endpoints[i], i
		}
	}
	return nil, -1
}

func AggAllEndpoints(clas []endpoint.ClusterLoadAssignment) []*endpoint.Endpoint {
	eps := make([]*endpoint.Endpoint, 0, len(clas))
	for i := range clas {
		for j := range clas[i].Endpoints {
			for x := range clas[i].Endpoints[j].LbEndpoints {
				if v, ok := clas[i].Endpoints[j].LbEndpoints[x].HostIdentifier.(*endpoint.LbEndpoint_Endpoint); ok {
					eps = append(eps, v.Endpoint)
				}
			}
		}
	}
	return eps
}

func ContainsEndpointAddr(addr string, clas []endpoint.ClusterLoadAssignment) bool {
	v, _ := FindEndpointAddr(addr, AggAllEndpoints(clas))
	return v != nil
}

func RemoveEndpointAddr(addr string, clas []endpoint.ClusterLoadAssignment) []endpoint.ClusterLoadAssignment {
	v, idx := FindEndpointAddr(addr, AggAllEndpoints(clas))
	if v != nil {
		return append(clas[:idx], clas[idx+1:]...)
	}
	return clas
}
