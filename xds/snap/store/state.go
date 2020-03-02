package store

import (
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

type EnvoyState struct {
	// The NodeID in the Envoy configuration. Required.
	Name string `json:"name"`

	// The Envoy Cluster definitions for the Envoy config. Required. See
	// https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/cluster.proto#cluster for more details.
	Clusters []api.Cluster `json:"clusters"`

	// The Envoy Route definitions for the Envoy config. Required. See
	// https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/route/route_components.proto#route-route for more details.
	Routes []route.Route `json:"routes"`

	// The Envoy Endpoint definitions for the Envoy config. Required. See
	// https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/endpoint/endpoint_components.proto#endpoint-endpoint for
	// more details.
	Endpoints []endpoint.Endpoint `json:"endpoints"`
}
