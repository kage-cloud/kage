package snap

import (
	"github.com/eddieowens/kage/xds/snap/store"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/google/uuid"
)

type StoreClient interface {
	Set(state *store.EnvoyState) error
	Get(nodeId string) (*store.EnvoyState, error)
	Load() error
}

type storeClient struct {
	Cache cache.SnapshotCache
	Store store.EnvoyStateStore
}

func (s *storeClient) Load() error {
	states, err := s.Store.FetchAll()
	if err != nil {
		return err
	}

	for _, state := range states {
		if err := s.Set(&state); err != nil {
			return err
		}
	}

	return nil
}

func (s *storeClient) Get(nodeId string) (*store.EnvoyState, error) {
	snap, err := s.Cache.GetSnapshot(nodeId)
	if err != nil {
		return nil, err
	}
	endpointsMap := snap.GetResources(cache.EndpointType)
	routesMap := snap.GetResources(cache.RouteType)
	clustersMap := snap.GetResources(cache.ClusterType)

	endpoints := make([]endpoint.Endpoint, len(endpointsMap))
	routes := make([]route.Route, len(routesMap))
	clusters := make([]api.Cluster, len(clustersMap))

	i := 0
	for _, v := range endpointsMap {
		re := v.(*endpoint.Endpoint)
		endpoints[i] = *re
	}

	i = 0
	for _, v := range routesMap {
		re := v.(*route.Route)
		routes[i] = *re
	}

	i = 0
	for _, v := range clustersMap {
		re := v.(*api.Cluster)
		clusters[i] = *re
	}

	return &store.EnvoyState{
		Name:      nodeId,
		Clusters:  clusters,
		Routes:    routes,
		Endpoints: endpoints,
	}, nil
}

func (s *storeClient) Set(state *store.EnvoyState) error {

	routeResources := make([]cache.Resource, len(state.Routes))
	clusterResources := make([]cache.Resource, len(state.Clusters))
	endpointResources := make([]cache.Resource, len(state.Endpoints))

	for i := range state.Routes {
		routeResources[i] = &state.Routes[i]
	}

	for i := range state.Endpoints {
		endpointResources[i] = &state.Endpoints[i]
	}

	for i := range state.Clusters {
		clusterResources[i] = &state.Clusters[i]
	}

	handler, err := s.Store.Save(state)
	if err != nil {
		return err
	}

	if err := s.Cache.SetSnapshot(state.Name, cache.NewSnapshot(
		uuid.New().String(),
		endpointResources,
		clusterResources,
		routeResources,
		nil,
		nil,
	)); err != nil {
		_ = handler.Revert()
		return err
	}

	return nil
}

func NewStoreClient() (StoreClient, error) {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	kubeStore, err := store.NewKubeStore()
	if err != nil {
		return nil, err
	}

	sc := &storeClient{
		Cache: snapshotCache,
		Store: kubeStore,
	}

	if err := sc.Load(); err != nil {
		return nil, err
	}

	return sc, nil
}
