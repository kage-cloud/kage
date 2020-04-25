package kubeutil

import (
	"github.com/kage-cloud/kage/core/kube/kengine"
	"k8s.io/client-go/util/workqueue"
)

func RemoveInformerIndex(s []kengine.InformEventHandler, index int) []kengine.InformEventHandler {
	return append(s[:index], s[index+1:]...)
}

func RemoveQueueIndex(s []workqueue.Interface, index int) []workqueue.Interface {
	return append(s[:index], s[index+1:]...)
}
