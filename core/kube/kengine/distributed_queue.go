package kengine

import (
	"context"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kinformer"
	"k8s.io/client-go/util/workqueue"
	"sync"
)

func NewDistributedQueue() DistributedQueue {
	return &distributedQueue{
		DelayingInterface: workqueue.NewDelayingQueue(),
		Queues:            make([]workqueue.Interface, 0),
		lock:              sync.RWMutex{},
	}
}

// A queue that can distribute its work amongst child queues.
type DistributedQueue interface {
	kinformer.FireAndForget
	workqueue.DelayingInterface
	AddQueue(queue workqueue.Interface)
}

type distributedQueue struct {
	workqueue.DelayingInterface
	Queues []workqueue.Interface
	lock   sync.RWMutex
}

func (d *distributedQueue) Shutdown() {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.DelayingInterface.ShutDown()
	for _, q := range d.Queues {
		q.ShutDown()
	}
}

func (d *distributedQueue) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		d.ShutDown()
	}()

	defer d.lock.RUnlock()
	shutdown := false
	for !shutdown {
		var item interface{}
		item, shutdown = d.Get()
		d.handle(item)
		d.pruneQueues()
	}
}

func (d *distributedQueue) handle(item interface{}) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	for _, q := range d.Queues {
		if !q.ShuttingDown() {
			q.Add(item)
		}
	}
}

func (d *distributedQueue) pruneQueues() {
	d.lock.Lock()
	defer d.lock.Unlock()
	toRemove := make([]int, 0)
	for i, q := range d.Queues {
		if q.ShuttingDown() {
			toRemove = append(toRemove, i)
		}
	}

	for _, v := range toRemove {
		d.Queues = kubeutil.RemoveQueueIndex(d.Queues, v)
	}
}

func (d *distributedQueue) AddQueue(queue workqueue.Interface) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.Queues = append(d.Queues, queue)
}
