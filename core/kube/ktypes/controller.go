package ktypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// Iterates through all controllers given an object. If false is returned, the walker stops, if an error is returned, it
// also stops but will return the error.
type ControllerWalker func(obj runtime.Object) (bool, error)

func GetLabelSelector(obj runtime.Object) labels.Selector {
	var selector labels.Selector
	switch typ := obj.(type) {
	case *corev1.Service:
		selector = labels.SelectorFromValidatedSet(typ.Spec.Selector)
	case *appsv1.Deployment:
		selector, _ = metav1.LabelSelectorAsSelector(typ.Spec.Selector)
	case *appsv1.StatefulSet:
		selector, _ = metav1.LabelSelectorAsSelector(typ.Spec.Selector)
	case *appsv1.ReplicaSet:
		selector, _ = metav1.LabelSelectorAsSelector(typ.Spec.Selector)
	case *appsv1.DaemonSet:
		selector, _ = metav1.LabelSelectorAsSelector(typ.Spec.Selector)
	}

	return selector
}
