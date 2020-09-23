package envoyutil

import route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

func AggAllRoutes(routeConfig []*route.RouteConfiguration) []*route.Route {
	routes := make([]*route.Route, len(routeConfig))
	for _, rc := range routeConfig {
		for _, vh := range rc.VirtualHosts {
			routes = append(routes, vh.Routes...)
		}
	}

	return routes
}
