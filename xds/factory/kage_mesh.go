package factory

import (
	"fmt"
	"github.com/eddieowens/kage/xds/model/consts"
	"github.com/eddieowens/kage/xds/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const KageMeshFactoryKey = "KageMeshFactory"

type KageMeshFactory interface {
	Deploy(targetDeployName string, replicaCount int32, ports []corev1.ContainerPort) *appsv1.Deployment
	BaselineConfigMap(targetDeployName string, content []byte) *corev1.ConfigMap
}

type kageMeshFactory struct {
}

func (k *kageMeshFactory) BaselineConfigMap(targetDeployName string, content []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-baseline-config", targetDeployName),
			Labels: map[string]string{
				consts.LabelKeyDomain:   consts.Domain,
				consts.LabelKeyResource: consts.LabelValueResourceKageMesh,
				consts.LabelKeyFor:      targetDeployName,
			},
		},
		BinaryData: map[string][]byte{
			consts.BaselineConfigMapFieldName: content,
		},
	}
}

func (k *kageMeshFactory) Deploy(targetDeployName string, replicaCount int32, ports []corev1.ContainerPort) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: util.GenKageMeshName(targetDeployName),
			Labels: map[string]string{
				consts.LabelKeyDomain:   consts.Domain,
				consts.LabelKeyResource: consts.LabelValueResourceKageMesh,
				consts.LabelKeyFor:      targetDeployName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(replicaCount),
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
}
