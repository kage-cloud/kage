package kubeutil

import (
	"github.com/kage-cloud/kage/core/kube/kengine"
	appsv1 "k8s.io/api/apps/v1"
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

func OwnerFilter(owners ...metav1.Object) kengine.Filter {
	return func(object metav1.Object) bool {
		return IsOwned(object, owners...)
	}
}

func SelectsDeployPodsFilter(deploys ...appsv1.Deployment) kengine.Filter {
	return func(object metav1.Object) bool {

		return false
	}
}

func FilterObject(filter kengine.Filter, objs ...metav1.Object) []metav1.Object {
	filteredObjs := make([]metav1.Object, 0)
	for _, o := range objs {
		if filter(o) {
			filteredObjs = append(filteredObjs, o)
		}
	}
	return filteredObjs
}

func ObjNames(objs []metav1.Object) []string {
	names := make([]string, len(objs))
	for i, o := range objs {
		names[i] = o.GetName()
	}
	return names
}
