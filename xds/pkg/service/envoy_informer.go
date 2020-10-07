package service

import (
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const EnvoyInformerServiceKey = "EnvoyInformerService"

type EnvoyInformerService interface {
	GetOrInitXds(canary metav1.Object) (*meta.Xds, error)
}

type envoyInformerService struct {
	EnvoyEndpointsService EnvoyEndpointsService `inject:"EnvoyEndpointsService"`
	KubeReaderService     KubeReaderService     `inject:"KubeReaderService"`
	KubeClient            kube.Client           `inject:"KubeClient"`
}

func (e *envoyInformerService) GetOrInitXds(canary metav1.Object) (*meta.Xds, error) {
	opt := kconfig.Opt{Namespace: canary.GetNamespace()}
	xdsAnno, _, err := e.getOrInitXdsAnno(canary, opt)
	return xdsAnno, err
}

func (e *envoyInformerService) storePods(pods []corev1.Pod, xdsAnno *meta.Xds, controllerType meta.ControllerType) error {
	for _, v := range pods {
		if err := e.EnvoyEndpointsService.StorePod(xdsAnno, controllerType, &v); err != nil {
			log.WithField("namespace", v.Namespace).
				WithField("name", v.Name).
				Error("Failed to add pod to mesh endpoints.")
		}
	}
	return nil
}

func (e *envoyInformerService) listControllersPods(controller runtime.Object, opt kconfig.Opt) ([]corev1.Pod, error) {
	selector := ktypes.GetLabelSelector(controller)

	return e.KubeReaderService.ListPods(selector, opt)
}

func (e *envoyInformerService) getOrInitXdsAnno(canary metav1.Object, opt kconfig.Opt) (*meta.Xds, runtime.Object, error) {
	controllerAnnos := canary.GetAnnotations()
	if controllerAnnos == nil {
		return nil, nil, except.NewError("canary has no annotations", except.ErrInvalid)
	}

	canaryAnno := new(meta.Canary)
	if err := meta.FromMap(controllerAnnos, canaryAnno); err != nil {
		return nil, nil, except.NewError("canary has unexpected annotation structure: %s", except.ErrInvalid, err.Error())
	}

	sourceAnno := canaryAnno.Source
	sourceKind := ktypes.Kind(sourceAnno.Kind)
	if !ktypes.IsController(sourceKind) {
		return nil, nil, except.NewError("the source is not a supported pod controller e.g. Deployment", except.ErrUnsupported)
	}

	source, err := e.KubeReaderService.Get(sourceAnno.Name, sourceKind, opt)
	if err != nil {
		return nil, nil, except.NewError("could not find source", except.ErrNotFound)
	}

	sourceMetaObj, ok := source.(metav1.Object)
	if !ok {
		return nil, nil, except.NewError("unsupported source kind", except.ErrUnsupported)
	}

	xdsAnno := new(meta.Xds)
	if err := meta.FromMap(controllerAnnos, xdsAnno); err != nil {
		return nil, nil, except.NewError("canary has unexpected annotation structure: %s", except.ErrInvalid, err.Error())
	}

	if xdsAnno.Config.NodeId == "" {
		e.instantiateXdsConfig(canaryAnno.Source.Name, xdsAnno)
		controllerAnnos = meta.Merge(controllerAnnos, xdsAnno)
		controllerAnnos = meta.Merge(controllerAnnos, &meta.XdsMarker{Type: meta.CanaryControllerType})

		canary.SetAnnotations(controllerAnnos)
		_, err = e.KubeClient.Update(canary.(runtime.Object), opt)
		if err != nil {
			return nil, nil, except.NewError("failed to update the canary: %s", except.ErrInvalid, err.Error())
		}

		sourceAnnos := sourceMetaObj.GetAnnotations()
		marker := meta.ToMap(&meta.XdsMarker{Type: meta.SourceControllerType})
		if !meta.Contains(sourceAnnos, marker) {
			sourceAnnos = meta.MergeMaps(sourceAnnos, marker)
			sourceAnnos = meta.Merge(sourceAnnos, xdsAnno)
			sourceMetaObj.SetAnnotations(sourceAnnos)
			_, err = e.KubeClient.Update(sourceMetaObj.(runtime.Object), opt)
			if err != nil {
				return nil, nil, except.NewError("failed to update the source's annotations: %s", except.ErrInvalid, err.Error())
			}
		}
	}

	return xdsAnno, source, nil
}

func (e *envoyInformerService) instantiateXdsConfig(controllerName string, xds *meta.Xds) {
	if xds.Config.Canary.ClusterName == "" {
		xds.Config.Canary.ClusterName = controllerName + "-canary"
	}

	if xds.Config.Source.ClusterName == "" {
		xds.Config.Canary.ClusterName = controllerName
	}

	if xds.Config.NodeId == "" {
		nodeId := uuid.New().String()
		xds.Config.NodeId = nodeId
	}
}
