package objconv

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func FromReplicaSets(rses []appsv1.ReplicaSet) []metav1.Object {
	rsObjs := make([]metav1.Object, len(rses))
	for i := range rses {
		rsObjs[i] = &rses[i]
	}
	return rsObjs
}

func ToReplicaSetsUnsafe(objs []metav1.Object) []appsv1.ReplicaSet {
	out := make([]appsv1.ReplicaSet, len(objs))
	for i := range objs {
		out[i] = *objs[i].(*appsv1.ReplicaSet)
	}
	return out
}
