package service

import (
	"github.com/eddieowens/axon"
	"github.com/eddieowens/kage/synchelpers"
	"sync"
)

const StopperHandlerServiceKey = "StopperHandlerService"

type StopperHandlerService interface {
	Add(key string, stopper synchelpers.Stopper)
	Remove(key string)
	Stop(key string, err error)
	Exists(key string) bool
}

type stopperHandlerService struct {
	lock          sync.RWMutex
	keyToStoppers map[string]synchelpers.Stopper
}

func (s *stopperHandlerService) Remove(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.keyToStoppers, key)
}

func (s *stopperHandlerService) Exists(key string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.keyToStoppers[key]
	return ok
}

func (s *stopperHandlerService) Add(key string, stopper synchelpers.Stopper) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.keyToStoppers[key] = stopper
}

func (s *stopperHandlerService) Stop(key string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	stopper := s.keyToStoppers[key]
	if stopper != nil {
		stopper.Stop(err)
	}
}

func stopHandlerFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&stopperHandlerService{
		lock:          sync.RWMutex{},
		keyToStoppers: map[string]synchelpers.Stopper{},
	})
}
