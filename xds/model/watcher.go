package model

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

type InformEventHandlerFuncs struct {
	OnWatch OnWatchEventFunc
	OnList  OnListEventFunc
}

func (i *InformEventHandlerFuncs) OnWatchEvent(event watch.Event) {
	if i.OnWatch != nil {
		i.OnWatch(event)
	}
}

func (i *InformEventHandlerFuncs) OnListEvent(obj runtime.Object) error {
	if i.OnList != nil {
		return i.OnList(obj)
	}
	return nil
}

type InformEventHandler interface {
	OnWatchEvent(event watch.Event)
	OnListEvent(obj runtime.Object) error
}

type OnWatchEventFunc func(event watch.Event)

type OnListEventFunc func(obj runtime.Object) error
