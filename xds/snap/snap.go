package snap

import (
	"fmt"
	"github.com/eddieowens/kage/xds/except"
	"github.com/eddieowens/kage/xds/snap/store"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/google/uuid"
	"sync"
)

const StoreClientKey = "StoreClient"

// Thread-safe client which owns and maintains the Envoy Snapshot cache. All EnvoyStates are backed up by a persistent
// storage and will be saved to persistent storage on every write. By default, the persistent storage are Kubernetes
// ConfigMaps.
type StoreClient interface {
	// Overwrite the current EnvoyState. For all unset fields on the EnvoyState, the previous EnvoyState will be used.
	Set(state *store.EnvoyState) error

	// Get the entirety of the EnvoyState.
	Get(nodeId string) (*store.EnvoyState, error)

	// Delete an EnvoyState with the specified name.
	Delete(nodeId string) error

	// Loads the entirety of the EnvoyState from the persistent Store into the StoreClient.
	Load() error

	// Reload a singular Node ID from the persistent store.
	Reload(nodeId string) error
}

type storeClient struct {
	// The Envoy Snapshot Cache which controls the current state of all Envoy sidecars in the cluster.
	Cache cache.SnapshotCache

	// The persistent storer for the EnvoyStates.
	Store store.EnvoyStateStore

	// EnvoyStates indexed by the node ID.
	CurrentStates map[string]store.EnvoyState

	syncChan chan string

	lock sync.RWMutex
}

func (s *storeClient) Reload(nodeId string) error {
	state, err := s.Store.Fetch(nodeId)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	currentState, _ := s.get(nodeId)

	if currentState != nil && state.UuidVersion == currentState.UuidVersion {
		return nil
	}

	return s.set(state)
}

func (s *storeClient) Stop() {
	close(s.syncChan)
}

func (s *storeClient) Start() error {
	s.syncChan = make(chan string)
	go func() {
		for nodeId := range s.syncChan {
			if err := s.Reload(nodeId); err != nil {
				fmt.Println("Failed to reload", nodeId, ":", err.Error())
			}
		}
	}()

	return nil
}

func (s *storeClient) Sync(nodeId string) error {
	s.syncChan <- nodeId
	return nil
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
		if err := s.set(&state); err != nil {
			return err
		}
	}

	return nil
}

func (s *storeClient) Get(nodeId string) (*store.EnvoyState, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.get(nodeId)
}

func (s *storeClient) get(nodeId string) (*store.EnvoyState, error) {
	v, ok := s.CurrentStates[nodeId]
	if !ok {
		return nil, except.NewError("node ID %s could not be found", except.ErrNotFound, nodeId)
	}
	return &v, nil
}

func (s *storeClient) Set(state *store.EnvoyState) error {
	s.lock.Lock()
	s.lock.Unlock()
	return s.set(state)
}

func (s *storeClient) set(state *store.EnvoyState) error {
	prevState, _ := s.get(state.NodeId)
	routes, routeResources := s.routes(prevState, state.Routes)
	listeners, listenerResources := s.listeners(prevState, state.Listeners)
	endpoints, endpointResources := s.endpoints(prevState, state.Endpoints)

	compositeState := &store.EnvoyState{
		NodeId:      state.NodeId,
		UuidVersion: uuid.New().String(),
		Listeners:   listeners,
		Routes:      routes,
		Endpoints:   endpoints,
	}

	handler, err := s.Store.Save(compositeState)
	if err != nil {
		return err
	}

	if err := s.Cache.SetSnapshot(state.NodeId, cache.NewSnapshot(
		compositeState.UuidVersion,
		endpointResources,
		nil,
		routeResources,
		listenerResources,
		nil,
	)); err != nil {
		_ = handler.Revert()
		return err
	}

	s.CurrentStates[state.NodeId] = *compositeState

	return nil
}

func (s *storeClient) routes(prevState *store.EnvoyState, routes []route.Route) ([]route.Route, []cache.Resource) {
	if len(routes) <= 0 && prevState != nil {
		routes = prevState.Routes
	}
	resources := make([]cache.Resource, len(routes))

	for i := range routes {
		resources[i] = &routes[i]
	}
	return routes, resources
}

func (s *storeClient) endpoints(prevState *store.EnvoyState, endpoints []endpoint.Endpoint) ([]endpoint.Endpoint, []cache.Resource) {
	if len(endpoints) <= 0 {
		endpoints = prevState.Endpoints
	}
	resources := make([]cache.Resource, len(endpoints))

	for i := range endpoints {
		resources[i] = &endpoints[i]
	}
	return endpoints, resources
}

func (s *storeClient) listeners(prevState *store.EnvoyState, listeners []api.Listener) ([]api.Listener, []cache.Resource) {
	if len(listeners) <= 0 {
		listeners = prevState.Listeners
	}
	resources := make([]cache.Resource, len(listeners))

	for i := range listeners {
		resources[i] = &listeners[i]
	}
	return listeners, resources
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
