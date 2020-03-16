package factory

import (
	"fmt"
	"github.com/eddieowens/kage/xds/model/consts"
	"github.com/eddieowens/kage/xds/util/canaryutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const KageMeshFactoryKey = "KageMeshFactory"

type KageMeshFactory interface {
	Deploy(name string, targetDeployName string, ports []corev1.ContainerPort, deployLabels, templateLabels map[string]string) *appsv1.Deployment
	BaselineConfigMap(targetDeployName string, content []byte) *corev1.ConfigMap
}

type kageMeshFactory struct {
}

func (k *kageMeshFactory) BaselineConfigMap(targetDeployName string, content []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-baseline-config", targetDeployName),
			Labels: canaryutil.GenKageMeshLabels(targetDeployName),
		},
		BinaryData: map[string][]byte{
			consts.BaselineConfigMapFieldName: content,
		},
	}
}

func (k *kageMeshFactory) Deploy(name string, targetDeployName string, ports []corev1.ContainerPort, deployLabels, templateLabels map[string]string) *appsv1.Deployment {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: canaryutil.GenKageMeshLabels(targetDeployName),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						consts.LabelKeyDomain:   consts.Domain,
						consts.LabelKeyResource: consts.LabelValueResourceKageMesh,
					},
					Annotations: map[string]string{
						consts.LabelKeyFor: targetDeployName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "envoyproxy/envoy:v1.13.1",
							Command: []string{
								"envoy", "-c", "/" + consts.BaselineConfigMapFieldName,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      consts.BaselineConfigMapName,
									ReadOnly:  true,
									MountPath: "/",
								},
							},
							Ports: ports,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: consts.BaselineConfigMapName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: consts.BaselineConfigMapName,
									},
									Optional: pointer.BoolPtr(false),
								},
							},
						},
					},
				},
			},
		},
	}

	for k, v := range deployLabels {
		dep.Labels[k] = v
	}
	for k, v := range templateLabels {
		dep.Spec.Template.Labels[k] = v
	}
	return dep
}
