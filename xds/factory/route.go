package factory

import (
	"fmt"
	envcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	envtype "github.com/envoyproxy/go-control-plane/envoy/type"
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

func (r *routeFactory) FromPercentage(endpointName string, percentage uint32) []route.Route {
	var servicePercentage uint32
	if percentage < 100 {
		servicePercentage = 100 - percentage
	}

	return []route.Route{
		{
			Name: fmt.Sprintf("%s-service", endpointName),
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_SafeRegex{
					SafeRegex: &matcher.RegexMatcher{
						EngineType: &matcher.RegexMatcher_GoogleRe2{
							GoogleRe2: &matcher.RegexMatcher_GoogleRE2{},
						},
						Regex: ".*",
					},
				},
				RuntimeFraction: &envcore.RuntimeFractionalPercent{
					DefaultValue: &envtype.FractionalPercent{
						Numerator:   servicePercentage,
						Denominator: envtype.FractionalPercent_HUNDRED,
					},
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: &route.WeightedCluster{
							Clusters: []*route.WeightedCluster_ClusterWeight{
								{
									Name:   fmt.Sprintf("%s-service", endpointName),
									Weight: &wrappers.UInt32Value{Value: servicePercentage},
								},
								{
									Name:   fmt.Sprintf("%s-canary", endpointName),
									Weight: &wrappers.UInt32Value{Value: percentage},
								},
							},
						},
					},
				},
			},
		},
	}
}
