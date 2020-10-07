package ktypes

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func SetObjects(li metav1.ListInterface, objs []runtime.Object) metav1.ListInterface {
	switch typ := li.(type) {
	case *corev1.ServiceList:
		objSli := make([]corev1.Service, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*corev1.Service); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *corev1.PodList:
		objSli := make([]corev1.Pod, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*corev1.Pod); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *appsv1.DeploymentList:
		objSli := make([]appsv1.Deployment, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*appsv1.Deployment); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *appsv1.StatefulSetList:
		objSli := make([]appsv1.StatefulSet, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*appsv1.StatefulSet); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *appsv1.DaemonSetList:
		objSli := make([]appsv1.DaemonSet, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*appsv1.DaemonSet); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *appsv1.ReplicaSetList:
		objSli := make([]appsv1.ReplicaSet, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*appsv1.ReplicaSet); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *corev1.ConfigMapList:
		objSli := make([]corev1.ConfigMap, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*corev1.ConfigMap); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *corev1.SecretList:
		objSli := make([]corev1.Secret, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*corev1.Secret); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	case *corev1.EndpointsList:
		objSli := make([]corev1.Endpoints, 0, len(objs))
		for _, v := range objs {
			if res, ok := v.(*corev1.Endpoints); ok {
				objSli = append(objSli, *res)
			}
		}
		typ.Items = objSli
		li = typ
	}

	return li
}
