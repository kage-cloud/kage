package factory

import (
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const KageMeshFactoryKey = "KageMeshFactory"

type KageMeshFactory interface {
	Deploy(name string, meshConfig *model.MeshConfig, ports []corev1.ContainerPort) *appsv1.Deployment
	BaselineConfigMap(name string, meshConfig *model.MeshConfig, content []byte) *corev1.ConfigMap
}

type kageMeshFactory struct {
}

func (k *kageMeshFactory) BaselineConfigMap(name string, meshConfig *model.MeshConfig, content []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: canaryutil.GenKageMeshLabels(meshConfig.Target.DeployName, meshConfig.Canary.DeployName),
			Annotations: canaryutil.GenKageMeshAnnotations(&model.KageMeshAnnotation{
				NodeId:            meshConfig.NodeId,
				CanaryClusterName: meshConfig.Canary.ClusterName,
				TargetClusterName: meshConfig.Target.ClusterName,
			}),
		},
		BinaryData: map[string][]byte{
			consts.BaselineConfigMapFieldName: content,
		},
	}
}

func (k *kageMeshFactory) Deploy(name string, meshConfig *model.MeshConfig, ports []corev1.ContainerPort) *appsv1.Deployment {
	labels := canaryutil.GenKageMeshLabels(meshConfig.Target.DeployName, meshConfig.Canary.DeployName)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: canaryutil.GenKageMeshAnnotations(&model.KageMeshAnnotation{
				NodeId:            meshConfig.NodeId,
				CanaryClusterName: meshConfig.Canary.ClusterName,
				TargetClusterName: meshConfig.Target.ClusterName,
			}),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
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
	return dep
}
