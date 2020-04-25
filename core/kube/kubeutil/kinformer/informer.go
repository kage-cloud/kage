package kinformer

import (
	"context"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kfilter"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

type FireAndForget interface {
	Start(ctx context.Context)
}

func RemoveInformerIndex(s []InformEventHandler, index int) []InformEventHandler {
	return append(s[:index], s[index+1:]...)
}

type InformerSpec struct {
	// The namespace and Kind that the informer watches.
	NamespaceKind ktypes.NamespaceKind

	BatchDuration time.Duration

	Filter kfilter.Filter

	Handlers []InformEventHandler
}

type InformEventHandlerFuncs struct {
	OnWatch OnWatchEventFunc
	OnList  OnListEventFunc
}

func (i *InformEventHandlerFuncs) OnWatchEvent(event watch.Event) error {
	if i.OnWatch != nil {
		return i.OnWatch(event)
	} else {
		return nil
	}
}

func (i *InformEventHandlerFuncs) OnListEvent(obj runtime.Object) error {
	if i.OnList != nil {
		return i.OnList(obj)
	}
	return nil
}

type InformEventHandler interface {
	OnWatchEvent(event watch.Event) error
	OnListEvent(obj runtime.Object) error
}

type OnWatchEventFunc func(event watch.Event) error

type OnListEventFunc func(obj runtime.Object) error