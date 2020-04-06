package store

import (
	"github.com/kage-cloud/kage/xds/except"
	"sync"
)

type memStore struct {
	lock sync.RWMutex
	m    map[string]EnvoyState
}

type memSaveHandler struct {
	store     *memStore
	prevState *EnvoyState
	currState *EnvoyState
}

func (m *memSaveHandler) Revert() error {
	if m.prevState == nil {
		return m.store.Delete(m.currState.NodeId)
	} else {
		_, err := m.store.Save(m.prevState)
		return err
	}
}

func (m *memStore) Save(state *EnvoyState) (SaveHandler, error) {
	m.lock.Lock()
	defer m.lock.Unlock()
	prev, ok := m.m[state.NodeId]
	var prevState *EnvoyState
	if ok {
		prevState = &prev
	}
	m.m[state.NodeId] = *state
	return &memSaveHandler{
		store:     m,
		prevState: prevState,
		currState: state,
	}, nil
}

func (m *memStore) Fetch(nodeId string) (*EnvoyState, error) {
	m.lock.RLock()
	m.lock.RUnlock()
	v, ok := m.m[nodeId]
	if !ok {
		return nil, except.NewError("Node ID %s could not be found", except.ErrNotFound, nodeId)
	}
	return &v, nil
}

func (m *memStore) FetchAll() ([]EnvoyState, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	states := make([]EnvoyState, len(m.m))
	i := 0
	for _, v := range m.m {
		states[i] = v
		i++
	}
	return states, nil
}

func (m *memStore) Delete(nodeId string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, ok := m.m[nodeId]
	if !ok {
		return except.NewError("Node ID %s could not be found", except.ErrNotFound, nodeId)
	}
	delete(m.m, nodeId)
	return nil
}

func NewInMemoryStore() EnvoyStatePersistentStore {
	return &memStore{
		lock: sync.RWMutex{},
		m:    map[string]EnvoyState{},
	}
}
