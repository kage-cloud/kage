package factory

import (
	"fmt"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/kage-cloud/kage/xds/pkg/model"
)

const RouteFactoryKey = "RouteFactory"

type RouteFactory interface {
	FromPercentage(meshConfig *model.MeshConfig) []*route.RouteConfiguration
}

func NewRouteFactory() RouteFactory {
	return new(routeFactory)
}

type routeFactory struct {
}

func (r *routeFactory) FromPercentage(meshConfig *model.MeshConfig) []*route.RouteConfiguration {
	name := fmt.Sprintf("%s-%s", meshConfig.Target.Name, meshConfig.Canary.Name)
	return []*route.RouteConfiguration{
		{
			Name: name,
			VirtualHosts: []*route.VirtualHost{
				{
					Name:    name,
					Domains: []string{""},
					Routes: []*route.Route{
						{
							Name: meshConfig.Target.Name,
							Match: &route.RouteMatch{
								PathSpecifier: &route.RouteMatch_Prefix{
									Prefix: "/",
								},
							},
							Action: &route.Route_Route{
								Route: &route.RouteAction{
									ClusterSpecifier: &route.RouteAction_WeightedClusters{
										WeightedClusters: &route.WeightedCluster{
											Clusters: []*route.WeightedCluster_ClusterWeight{
												{
													Name:   meshConfig.Target.Name,
													Weight: &wrappers.UInt32Value{Value: meshConfig.Target.RoutingWeight},
												},
												{
													Name:   meshConfig.Canary.Name,
													Weight: &wrappers.UInt32Value{Value: meshConfig.Canary.RoutingWeight},
												},
											},
											TotalWeight: &wrappers.UInt32Value{Value: meshConfig.TotalRoutingWeight},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
