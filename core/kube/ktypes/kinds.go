package ktypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Kind string

const (
	KindUnknown     Kind = ""
	KindPod         Kind = "Pod"
	KindDeployment  Kind = "Deployment"
	KindService     Kind = "Service"
	KindReplicaSet  Kind = "ReplicaSet"
	KindConfigMap   Kind = "ConfigMap"
	KindEndpoints   Kind = "Endpoints"
	KindStatefulSet Kind = "StatefulSet"
	KindDaemonSet   Kind = "DaemonSet"
)

// Returns true if the Kind is a controller. A Controller is a Kube resource that controls Pods, e.g. Deploy, ReplicaSets,
// StatefulSets, etc. Pods are also considered Controllers.
func IsController(k Kind) bool {
	switch k {
	case KindDeployment, KindReplicaSet, KindPod, KindStatefulSet, KindDaemonSet:
		return true
	}
	return false
}

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
		return KindDeployment
	case *appsv1.ReplicaSet:
		return KindReplicaSet
	case *corev1.ConfigMap:
		return KindConfigMap
	case *corev1.Endpoints:
		return KindEndpoints
	case *appsv1.StatefulSet:
		return KindStatefulSet
	case *appsv1.DaemonSet:
		return KindDaemonSet
	}
	return KindUnknown
}
