package service

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const CanaryServiceKey = "CanaryService"

type CanaryService interface {
	FetchForPod(pod *corev1.Pod) *meta.Canary
	FetchForController(obj runtime.Object) *meta.Canary
}

type canaryService struct {
	KubeReaderService KubeReaderService `inject:"KubeReaderService"`
	KubeClient        kube.Client       `inject:"KubeClient"`
}

func (c *canaryService) FetchForController(obj runtime.Object) *meta.Canary {
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil
	}

	canary, err := c.unmarshalAnno(metaObj)
	if err != nil {
		return nil
	}

	return canary
}

func (c *canaryService) FetchForPod(pod *corev1.Pod) *meta.Canary {
	return c.getCanaryAnnoForPod(pod)
}

func (c *canaryService) getCanaryAnnoForPod(pod *corev1.Pod) *meta.Canary {
	var canaryAnno *meta.Canary
	var err error
	_ = c.KubeReaderService.WalkControllers(pod, func(controller runtime.Object) (bool, error) {
		metaObj, ok := controller.(metav1.Object)
		if !ok {
			return true, nil
		}

		canaryAnno, err = c.unmarshalAnno(metaObj)
		if err != nil {
			return true, nil
		}

		if canaryAnno.SourceObj.Name != "" {
			return false, nil
		}

		return true, nil
	})

	return canaryAnno
}

func (c *canaryService) unmarshalAnno(metaObj metav1.Object) (*meta.Canary, error) {
	canaryAnno := new(meta.Canary)
	if err := meta.FromMap(metaObj.GetAnnotations(), canaryAnno); err != nil {
		return nil, err
	}

	canaryAnno.CanaryObj = meta.ObjRef{
		Name:      metaObj.GetName(),
		Kind:      string(ktypes.KindFromObject(metaObj.(runtime.Object))),
		Namespace: metaObj.GetNamespace(),
	}

	return canaryAnno, nil
}
