package service

import (
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/kage-cloud/kage/xds/except"
	"github.com/kage-cloud/kage/xds/snap/snaputil"
	"github.com/kage-cloud/kage/xds/snap/store"
)

const EnvoyStateServiceKey = "EnvoyStateService"

type EnvoyStateService interface {
	// Safely finds the canary's weighted traffic routing. If the weight is not available, an error is returned.
	FetchCanaryRouteWeight(state *store.EnvoyState) (uint32, error)
}

type envoystateService struct {
}

func (e *envoystateService) FetchCanaryRouteWeight(state *store.EnvoyState) (uint32, error) {
	for _, r := range state.Routes {
		action, ok := r.Action.(*route.Route_Route)
		if ok {
			if action.Route != nil {
				cs, ok := action.Route.ClusterSpecifier.(*route.RouteAction_WeightedClusters)
				if ok {
					if cs.WeightedClusters != nil {
						for _, cluster := range cs.WeightedClusters.Clusters {
							if cluster != nil && snaputil.IsCanaryName(cluster.Name) && cluster.Weight != nil {
								return cluster.Weight.Value, nil
							}
						}
					}
				}
			}
		}
	}
	return 0, except.NewError("No canary routes found for %s", except.ErrNotFound, state.NodeId)
}
