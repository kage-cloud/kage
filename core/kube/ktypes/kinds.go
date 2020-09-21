package ktypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Kind string

const (
	KindUnknown    Kind = ""
	KindPod        Kind = "Pod"
	KindDeploy     Kind = "Deploy"
	KindService    Kind = "Service"
	KindReplicaSet Kind = "ReplicaSet"
	KindConfigMap  Kind = "ConfigMap"
	KindEndpoints  Kind = "Endpoints"
)

func IsKind(k Kind, obj runtime.Object) bool {
	return KindFromObject(obj) == k
}

func KindFromObject(obj runtime.Object) Kind {
	switch obj.(type) {
	case *corev1.Pod:
		return KindPod
	case *corev1.Service:
		return KindService
	case *appsv1.Deployment:
		return KindDeploy
	case *appsv1.ReplicaSet:
		return KindReplicaSet
	case *corev1.ConfigMap:
		return KindConfigMap
	case *corev1.Endpoints:
		return KindEndpoints
	}
	return KindUnknown
}
