package factory

import (
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/snap/snaputil"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes/wrappers"
)

const RouteFactoryKey = "RouteFactory"

type RouteFactory interface {
	FromPercentage(endpointName string, percentage uint32) []route.Route
}

func NewRouteFactory() RouteFactory {
	return new(routeFactory)
}

type routeFactory struct {
}

func (r *routeFactory) FromPercentage(endpointsName string, percentage uint32) []route.Route {
	var servicePercentage uint32
	if percentage < model.TotalRoutingWeight {
		servicePercentage = model.TotalRoutingWeight - percentage
	}

	return []route.Route{
		{
			Name: snaputil.GenServiceName(endpointsName),
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_SafeRegex{
					SafeRegex: &matcher.RegexMatcher{
						EngineType: &matcher.RegexMatcher_GoogleRe2{
							GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
						},
						Regex: ".*",
					},
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: &route.WeightedCluster{
							Clusters: []*route.WeightedCluster_ClusterWeight{
								{
									Name:   snaputil.GenServiceName(endpointsName),
									Weight: &wrappers.UInt32Value{Value: servicePercentage},
								},
								{
									Name:   snaputil.GenCanaryName(endpointsName),
									Weight: &wrappers.UInt32Value{Value: percentage},
								},
							},
							TotalWeight: &wrappers.UInt32Value{Value: model.TotalRoutingWeight},
						},
					},
				},
			},
		},
	}
}
