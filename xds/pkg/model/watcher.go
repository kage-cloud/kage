package model

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type InformEventHandlerFuncs struct {
	OnWatch OnWatchEventFunc
	OnList  OnListEventFunc
}

func (i *InformEventHandlerFuncs) OnWatchEvent(event watch.Event) bool {
	if i.OnWatch != nil {
		return i.OnWatch(event)
	} else {
		return false
	}
}

func (i *InformEventHandlerFuncs) OnListEvent(obj runtime.Object) error {
	if i.OnList != nil {
		return i.OnList(obj)
	}
	return nil
}

type InformEventHandler interface {
	OnWatchEvent(event watch.Event) bool
	OnListEvent(obj runtime.Object) error
}

type OnWatchEventFunc func(event watch.Event) bool

type OnListEventFunc func(obj runtime.Object) error
