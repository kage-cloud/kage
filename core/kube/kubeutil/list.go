package kubeutil

import (
	"github.com/kage-cloud/kage/core/kube/ktypes"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func ToListType(kind ktypes.Kind, objs []runtime.Object) metav1.ListInterface {
	var li metav1.ListInterface
	switch kind {
	case ktypes.KindPod:
		items := make([]corev1.Pod, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*corev1.Pod); ok {
				items = append(items, *v)
			}
		}
		li = &corev1.PodList{
			Items: items,
		}
	case ktypes.KindDeployment:
		items := make([]appsv1.Deployment, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*appsv1.Deployment); ok {
				items = append(items, *v)
			}
		}
		li = &appsv1.DeploymentList{
			Items: items,
		}
	case ktypes.KindService:
		items := make([]corev1.Service, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*corev1.Service); ok {
				items = append(items, *v)
			}
		}
		li = &corev1.ServiceList{
			Items: items,
		}
	case ktypes.KindReplicaSet:
		items := make([]appsv1.ReplicaSet, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*appsv1.ReplicaSet); ok {
				items = append(items, *v)
			}
		}
		li = &appsv1.ReplicaSetList{
			Items: items,
		}
	case ktypes.KindConfigMap:
		items := make([]corev1.ConfigMap, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*corev1.ConfigMap); ok {
				items = append(items, *v)
			}
		}
		li = &corev1.ConfigMapList{
			Items: items,
		}
	case ktypes.KindEndpoints:
		items := make([]corev1.Endpoints, 0, len(objs))
		for _, o := range objs {
			if v, ok := o.(*corev1.Endpoints); ok {
				items = append(items, *v)
			}
		}
		li = &corev1.EndpointsList{
			Items: items,
		}
	}

	return li
}

func ObjectsFromList(list metav1.ListInterface) []runtime.Object {
	var objs []runtime.Object
	switch typ := list.(type) {
	case *appsv1.DeploymentList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *appsv1.StatefulSetList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *appsv1.ReplicaSetList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *appsv1.DaemonSetList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.PodList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.ConfigMapList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.SecretList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.EndpointsList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.ServiceList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.NamespaceList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.NodeList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.PersistentVolumeList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.PersistentVolumeClaimList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.ComponentStatusList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.ServiceAccountList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *corev1.PodTemplateList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	case *batchv1beta1.CronJobList:
		objs = make([]runtime.Object, 0, len(typ.Items))
		for i := range typ.Items {
			objs = append(objs, &typ.Items[i])
		}
	}
	return objs
}
