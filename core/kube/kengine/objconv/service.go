package objconv

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromServices(svcs []corev1.Service) []metav1.Object {
	depObjs := make([]metav1.Object, len(svcs))
	for i := range svcs {
		depObjs[i] = &svcs[i]
	}
	return depObjs
}
