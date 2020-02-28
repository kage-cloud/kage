package snap

import (
	"github.com/eddieowens/kage/xds/snap/store"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/google/uuid"
)

type StoreClient interface {
	Set(state *store.EnvoyState) error
}

type storeClient struct {
	Cache cache.SnapshotCache
	Store store.EnvoyStateStore
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
}

func NewStoreClient() (StoreClient, error) {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
}
