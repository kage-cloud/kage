package objconv

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromDeployments(deploys []appsv1.Deployment) []metav1.Object {
	depObjs := make([]metav1.Object, len(deploys))
	for i := range deploys {
		depObjs[i] = &deploys[i]
	}
	return depObjs
}
