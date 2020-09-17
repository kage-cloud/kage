package store

import (
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/proto"
	"time"
)

type EnvoyState struct {
	proto.Message

	// The NodeID in the Envoy configuration. Required.
	NodeId string `json:"node_id"`

	// A unique version delegated to every single Envoy State.
	UuidVersion string `json:"uuid_version"`

	// The time of creation in UTC.
	CreationTimestampUtc time.Time `json:"creation_timestamp_utc"`

	// The Envoy Listener definitions for the Envoy config. Required.
	Listeners []listener.Listener `json:"listeners"`

	// The Envoy Route definitions for the Envoy config. Required.
	Routes []route.Route `json:"routes"`

	// The Envoy Endpoint definitions for the Envoy config. Required.
	// more details.
	Endpoints []endpoint.Endpoint `json:"endpoints"`
}
