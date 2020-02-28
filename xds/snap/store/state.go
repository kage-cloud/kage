package store

import (
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

type EnvoyState struct {
	Name      string              `json:"name"`
	Clusters  []api.Cluster       `json:"clusters"`
	Routes    []route.Route       `json:"routes"`
	Endpoints []endpoint.Endpoint `json:"endpoints"`
}
