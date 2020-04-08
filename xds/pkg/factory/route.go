package factory

import (
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/kage-cloud/kage/xds/pkg/model"
)

const RouteFactoryKey = "RouteFactory"

type RouteFactory interface {
	FromPercentage(canaryName, serviceName string, canaryPercentage, servicePercentage uint32) []route.Route
}

func NewRouteFactory() RouteFactory {
	return new(routeFactory)
}

type routeFactory struct {
}

func (r *routeFactory) FromPercentage(canaryName, serviceName string, canaryPercentage, servicePercentage uint32) []route.Route {
	return []route.Route{
		{
			Name: serviceName,
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
									Name:   serviceName,
									Weight: &wrappers.UInt32Value{Value: servicePercentage},
								},
								{
									Name:   canaryName,
									Weight: &wrappers.UInt32Value{Value: canaryPercentage},
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
