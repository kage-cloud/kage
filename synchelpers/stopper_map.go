package synchelpers

import "sync"

// Centralized and thread-safe storage for your stoppers.
type StopperMap interface {
	Add(key string, stopper Stopper)
	Remove(key string)
	Stop(key string, err error)
	Exists(key string) bool
}

func NewStopperMap() StopperMap {
	return &stopperMap{
		lock:          sync.RWMutex{},
		keyToStoppers: map[string]Stopper{},
	}
}

func (s *stopperMap) Remove(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.keyToStoppers, key)
}

func (s *stopperMap) Exists(key string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.keyToStoppers[key]
	return ok
}

func (s *stopperMap) Add(key string, stopper Stopper) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.keyToStoppers[key] = stopper
}

func (s *stopperMap) Stop(key string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	stopper := s.keyToStoppers[key]
	if stopper != nil {
		stopper.Stop(err)
	}
}

type stopperMap struct {
	lock          sync.RWMutex
	keyToStoppers map[string]Stopper
}
