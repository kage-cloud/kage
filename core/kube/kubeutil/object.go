package kubeutil

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsOwned(owned metav1.Object, owners ...metav1.Object) bool {
	for _, owner := range owners {
		for _, or := range owned.GetOwnerReferences() {
			if or.Name == owner.GetName() && owner.GetNamespace() == owned.GetNamespace() {
				return true
			}
		}
	}
	return false
}

func ObjNames(objs []metav1.Object) []string {
	names := make([]string, len(objs))
	for i, o := range objs {
		names[i] = o.GetName()
	}
	return names
}
