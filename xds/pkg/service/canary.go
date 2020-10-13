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
}

type canaryService struct {
	KubeReaderService KubeReaderService `inject:"KubeReaderService"`
	KubeClient        kube.Client       `inject:"KubeClient"`
}

func (c *canaryService) FetchForPod(pod *corev1.Pod) *meta.Canary {
	return c.getCanaryAnnoForPod(pod)
}

func (c *canaryService) getCanaryAnnoForPod(pod *corev1.Pod) *meta.Canary {
	var canaryAnno *meta.Canary
	_ = c.KubeReaderService.WalkControllers(pod, func(controller runtime.Object) (bool, error) {
		metaObj, ok := controller.(metav1.Object)
		if !ok {
			return true, nil
		}
		annos := metaObj.GetAnnotations()

		if err := meta.FromMap(annos, canaryAnno); err != nil {
			return true, nil
		}

		if canaryAnno.SourceObj.Name != "" {
			return false, nil
		}

		canaryAnno.CanaryObj = meta.ObjRef{
			Name:      metaObj.GetName(),
			Kind:      string(ktypes.KindFromObject(controller)),
			Namespace: metaObj.GetNamespace(),
		}

		return true, nil
	})

	return canaryAnno
}
