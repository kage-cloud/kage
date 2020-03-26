package service

import (
	"fmt"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/synchelpers"
	"github.com/kage-cloud/kage/xds/model/consts"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/snap/snaputil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
)

const StateSyncServiceKey = "StateSyncService"

type StateSyncService interface {
	Start() (synchelpers.Stopper, error)
}

type stateSyncService struct {
	KubeClient  kube.Client      `inject:"KubeClient"`
	StoreClient snap.StoreClient `inject:"StoreClient"`
}

func (c *stateSyncService) Start() (synchelpers.Stopper, error) {
	selector := labels.SelectorFromSet(map[string]string{
		consts.LabelKeyResource: consts.LabelValueResourceSnapshot,
	})

	_, event := c.KubeClient.InformAndListConfigMap(func(object v1.Object) bool {
		return selector.Matches(labels.Set(object.GetLabels()))
	})

	onStop := make(chan error)

	go func() {
		for {
			select {
			case e := <-event:
				switch e.Type {
				case watch.Modified, watch.Added:
					if cm, ok := e.Object.(*corev1.ConfigMap); ok {
						for _, nodeId := range snaputil.NodeIdsFromConfigMap(cm) {
							if err := c.StoreClient.Reload(nodeId); err != nil {
								fmt.Println("Failed to reload node ID", nodeId)
							}
						}
					}
				}
			case err := <-onStop:
				if err != nil {
					fmt.Println("Stopping syncer due to error:", err.Error())
					return
				}
			}
		}
	}()

	return synchelpers.NewStopper(func(err error) {
		onStop <- err
	}), nil
}
