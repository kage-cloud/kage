package kubeinformer

import (
	"context"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/service"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

var controllers = []ktypes.Kind{
	ktypes.KindStatefulSet,
	ktypes.KindDaemonSet,
	ktypes.KindDeployment,
	ktypes.KindPod,
	ktypes.KindReplicaSet,
}

type Canary struct {
	InformerClient  kube.InformerClient     `inject:"InformerClient"`
	CanaryService   service.CanaryService   `inject:"CanaryService"`
	KageMeshService service.KageMeshService `inject:"KageMeshService"`
}

func (c *Canary) Inform(ctx context.Context) error {
	informerSpec := kinformer.InformerSpec{
		BatchDuration: 15 * time.Second,
		Filter:        kfilter.LabelSelectorFilter(labels.SelectorFromValidatedSet(meta.ToMap(&meta.CanaryMarker{Canary: true}))),
		Handlers:      []kinformer.InformEventHandler{c.canaryEventHandler()},
	}

	for _, controller := range controllers {
		informerSpec.NamespaceKind = ktypes.NamespaceKind{Kind: controller}
		if err := c.InformerClient.Inform(ctx, informerSpec); err != nil {
			return err
		}
	}

	return nil
}

func (c *Canary) canaryEventHandler() kinformer.InformEventHandler {
	return &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			switch event.Type {
			case watch.Deleted, watch.Error:
				c.deleteCanary(event.Object)
			}
			return nil
		},
	}
}

func (c *Canary) deleteCanary(obj runtime.Object) {
	canary := c.CanaryService.FetchForController(obj)
	if canary == nil {
		metaObj, ok := obj.(metav1.Object)
		if ok {
			logrus.WithField("name", metaObj.GetName()).
				WithField("namespace", metaObj.GetNamespace()).
				Debug("Canary deleted but did not have a valid annotation.")
		}
		return
	}

	kageProxyAnno, err := c.KageMeshService.FetchForCanary(canary)
	if err != nil {
		logrus.WithField("name", canary.CanaryObj.Name).
			WithField("namespace", canary.CanaryObj.Namespace).
			WithError(err).
			Error("Failed to fetch kage proxy for canary after it was deleted.")
		return
	}

	opt := kconfig.Opt{Namespace: canary.CanaryObj.Namespace}
	if err := c.KageMeshService.Remove(kageProxyAnno, opt); err != nil {
		logrus.WithField("name", canary.CanaryObj.Name).
			WithField("namespace", canary.CanaryObj.Namespace).
			WithField("proxy_name", kageProxyAnno.Name).
			WithError(err).
			Error("Failed to delete kage proxy for canary after it was deleted.")
	}
}
