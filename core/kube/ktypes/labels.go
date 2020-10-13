package ktypes

import "k8s.io/apimachinery/pkg/labels"

func UnionSet(s1, s2 labels.Set) labels.Set {
	if len(s1) == 0 {
		return s2
	}
	if len(s2) == 0 {
		return s1
	}
	aggSet := labels.Set{}
	for k, v := range s1 {
		if s2.Has(k) {
			aggSet[k] = v
		}
	}

	for k, v := range s2 {
		if s1.Has(k) {
			aggSet[k] = v
		}
	}

	return aggSet
}
