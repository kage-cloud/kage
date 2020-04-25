package service

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kfilter"
	"github.com/kage-cloud/kage/core/kube/kubeutil/kinformer"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const StateSyncServiceKey = "StateSyncService"

type StateSyncService interface {
	Start() error
}

type stateSyncService struct {
	InformerClient kube.InformerClient `inject:"InformerClient"`
	StoreClient    snap.StoreClient    `inject:"StoreClient"`
	Config         *config.Config      `inject:"Config"`
}

func (c *stateSyncService) Start() error {
	selector := labels.SelectorFromSet(map[string]string{
		consts.LabelKeyResource: consts.LabelValueResourceSnapshot,
	})

	spec := kinformer.InformerSpec{
		NamespaceKind: ktypes.NewNamespaceKind(c.Config.Kube.Namespace, ktypes.KindConfigMap),
		BatchDuration: 1 * time.Second,
		Filter:        kfilter.SelectedObjectFilter(selector),
		Handlers: []kinformer.InformEventHandler{
			&kinformer.InformEventHandlerFuncs{
				OnWatch: func(event watch.Event) error {
					switch event.Type {
					case watch.Modified, watch.Added:
						if cm, ok := event.Object.(*corev1.ConfigMap); ok {
							log.WithField("name", cm.Name).
								WithField("namespace", cm.Namespace).
								Debug("Detected a change in the Envoy config ConfigMap. Reloading.")
							for _, nodeId := range snaputil.NodeIdsFromConfigMap(cm) {
								if err := c.StoreClient.Reload(nodeId); err != nil {
									log.WithField("name", cm.Name).
										WithField("namespace", cm.Namespace).
										WithField("node_id", nodeId).
										WithError(err).
										Error("Failed to reload Envoy config from ConfigMap.")
									return err
								}
							}
							log.WithField("name", cm.Name).
								WithField("namespace", cm.Namespace).
								Debug("Reloaded Envoy config from ConfigMap.")
						}
					}
					return nil
				},
			},
		},
	}

	return c.InformerClient.Inform(context.Background(), spec)
}
