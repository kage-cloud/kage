package snap

import (
	"fmt"
	"github.com/eddieowens/kage/xds/except"
	"github.com/eddieowens/kage/xds/snap/store"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/google/uuid"
	"sync"
)

// Thread-safe client which owns and maintains the Envoy Snapshot cache. All EnvoyStates are backed up by a persistent
// storage and will be saved to persistent storage on every write. By default, the persistent storage are Kubernetes
// ConfigMaps.
type StoreClient interface {
	// Overwrite the entirety of the EnvoyState.
	Set(state *store.EnvoyState) error

	// Get the entirety of the EnvoyState.
	Get(nodeId string) (*store.EnvoyState, error)

	// Delete an EnvoyState with the specified name.
	Delete(nodeId string) error

	// Loads the entirety of the EnvoyState from the persistent Store into the StoreClient.
	Load() error
}

type storeClient struct {
	// The Envoy Snapshot Cache which controls the current state of all Envoy sidecars in the cluster.
	Cache cache.SnapshotCache

	// The persistent storer for the EnvoyStates.
	Store store.EnvoyStateStore

	// EnvoyStates indexed by the node ID.
	CurrentStates map[string]store.EnvoyState

	lock sync.RWMutex
}

func (s *storeClient) Delete(nodeId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.Store.Delete(nodeId); err != nil {
		fmt.Printf("Failed to delete %s from the persistent store: %s", nodeId, err.Error())
	}

	s.Cache.ClearSnapshot(nodeId)

	delete(s.CurrentStates, nodeId)

	return nil
}

func (s *storeClient) Load() error {
	states, err := s.Store.FetchAll()
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, state := range states {
		if err := s.Set(&state); err != nil {
			return err
		}
	}

	return nil
}

func (s *storeClient) Get(nodeId string) (*store.EnvoyState, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	v, ok := s.CurrentStates[nodeId]
	if !ok {
		return nil, except.NewError("node ID %s could not be found", except.ErrNotFound, nodeId)
	}
	return &v, nil
}

func (s *storeClient) Set(state *store.EnvoyState) error {
	s.lock.Lock()
	s.lock.Unlock()
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

	s.CurrentStates[state.Name] = *state

	return nil
}

// Create a new StoreClient to save the EnvoyStates and update the Envoy Snapshot cache.
func NewStoreClient() (StoreClient, error) {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	kubeStore, err := store.NewKubeStore()
	if err != nil {
		return nil, err
	}

	sc := &storeClient{
		Cache:         snapshotCache,
		Store:         kubeStore,
		CurrentStates: map[string]store.EnvoyState{},
		lock:          sync.RWMutex{},
	}

	if err := sc.Load(); err != nil {
		return nil, err
	}

	return sc, nil
}
