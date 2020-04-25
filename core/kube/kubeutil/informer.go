package kubeutil

import (
	"k8s.io/client-go/util/workqueue"
)

func RemoveQueueIndex(s []workqueue.Interface, index int) []workqueue.Interface {
	return append(s[:index], s[index+1:]...)
}
