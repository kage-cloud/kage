package factory

import (
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"path"
)

const KageMeshFactoryKey = "KageMeshFactory"

type KageMeshFactory interface {
	Deploy(name, canaryDeployName, targetDeployName string, meshConfig *model.MeshConfig, ports []corev1.ContainerPort) *appsv1.Deployment
	BaselineConfigMap(name, canaryDeployName, targetDeployName string, meshConfig *model.MeshConfig, content []byte) *corev1.ConfigMap
}

type kageMeshFactory struct {
}

func (k *kageMeshFactory) BaselineConfigMap(name, canaryDeployName, targetDeployName string, meshConfig *model.MeshConfig, content []byte) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: canaryutil.GenKageMeshLabels(targetDeployName, canaryDeployName),
			Annotations: canaryutil.GenKageMeshAnnotations(&model.MeshConfigAnnotation{
				NodeId:            meshConfig.NodeId,
				CanaryClusterName: meshConfig.Canary.Name,
				TargetClusterName: meshConfig.Target.Name,
			}),
		},
		Data: map[string]string{
			consts.BaselineConfigMapFieldName: string(content),
		},
	}
}

func (k *kageMeshFactory) Deploy(name, canaryDeployName, targetDeployName string, meshConfig *model.MeshConfig, ports []corev1.ContainerPort) *appsv1.Deployment {
	labels := canaryutil.GenKageMeshLabels(targetDeployName, canaryDeployName)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: canaryutil.GenKageMeshAnnotations(&model.MeshConfigAnnotation{
				NodeId:            meshConfig.NodeId,
				CanaryClusterName: meshConfig.Canary.Name,
				TargetClusterName: meshConfig.Target.Name,
			}),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: metav1.SetAsLabelSelector(labels),
			Replicas: pointer.Int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "kage-mesh",
							Image: "envoyproxy/envoy:v1.15.0",
							Command: []string{
								"envoy", "-c", path.Join("/etc/envoy", consts.BaselineConfigMapFieldName),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      consts.BaselineConfigMapName,
									ReadOnly:  true,
									MountPath: "/etc/envoy",
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
										Name: name,
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
