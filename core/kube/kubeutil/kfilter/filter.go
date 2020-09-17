package kfilter

import (
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type Filter func(object metav1.Object) bool

func FilterObject(filter Filter, objs ...metav1.Object) []metav1.Object {
	filteredObjs := make([]metav1.Object, 0)
	for _, o := range objs {
		if filter(o) {
			filteredObjs = append(filteredObjs, o)
		}
	}
	return filteredObjs
}

func OwnerFilter(owners ...metav1.Object) Filter {
	return func(object metav1.Object) bool {
		return kubeutil.IsOwned(object, owners...)
	}
}

// selects all object that match at least one of the selectors
func LazyMatchesSelectorsFilter(selectors ...labels.Selector) Filter {
	return func(object metav1.Object) bool {
		matches := false
		for i := 0; i < len(selectors) && !matches; i++ {
			matches = selectors[i].Matches(labels.Set(object.GetLabels()))
		}
		return matches
	}
}

func SelectedObjectFilter(selector labels.Selector) Filter {
	return func(object metav1.Object) bool {
		return selector.Matches(labels.Set(object.GetLabels()))
	}
}

func SelectsDeployPodsFilter(deploys ...appsv1.Deployment) Filter {
	return func(object metav1.Object) bool {

		return false
	}
}
