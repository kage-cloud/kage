package kengine

import (
	"context"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/workqueue"
)

func NewHandlerQueue(handlers ...kinformer.InformEventHandler) HandlerQueue {
	return &handlerQueue{
		RateLimitingInterface: workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		Handlers:              handlers,
	}
}

type HandlerQueue interface {
	kinformer.FireAndForget
	workqueue.RateLimitingInterface
}

type handlerQueue struct {
	workqueue.RateLimitingInterface
	Handlers []kinformer.InformEventHandler
}

func (h *handlerQueue) Start(ctx context.Context) {
	go h.start()
	go func() {
		<-ctx.Done()
		h.ShutDown()
	}()
}

func (h *handlerQueue) start() {
	shutdown := false
	for !shutdown {
		var item interface{}
		item, shutdown = h.Get()
		if v, ok := item.(watch.Event); ok {
			h.Handle(v)
		}
	}
}

func (h *handlerQueue) Handle(event watch.Event) {
	var isDone bool
	for _, handler := range h.Handlers {
		err := handler.OnWatchEvent(event)
		isDone = isDone && err == nil
	}
	if isDone {
		h.Done(event)
	}
}
