package synchelpers

import (
	"context"
	"sync"
)

type CancelFuncMap interface {
	Set(key string, cancelFuncs ...context.CancelFunc)
	Add(key string, cancelFuncs ...context.CancelFunc)
	Remove(key string)
	Cancel(key string)
	Exists(key string) bool
}

func NewCancelFuncMap() CancelFuncMap {
	return &cancelFuncMap{
		m:    map[string][]context.CancelFunc{},
		lock: sync.RWMutex{},
	}
}

type cancelFuncMap struct {
	m    map[string][]context.CancelFunc
	lock sync.RWMutex
}

func (c *cancelFuncMap) Set(key string, cancelFuncs ...context.CancelFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.m[key] = cancelFuncs
}

func (c *cancelFuncMap) Add(key string, cancelFuncs ...context.CancelFunc) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if _, ok := c.m[key]; !ok {
		c.m[key] = []context.CancelFunc{}
	}
	c.m[key] = append(c.m[key], cancelFuncs...)
}

func (c *cancelFuncMap) Remove(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.m, key)
}

func (c *cancelFuncMap) Cancel(key string) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if v, ok := c.m[key]; ok {
		for _, cf := range v {
			cf()
		}
	}
}

func (c *cancelFuncMap) Exists(key string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.m[key]
	return ok
}
