package model

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type MeshConfigAnnotation struct {
	NodeId            string
	CanaryClusterName string
	TargetClusterName string
}

type KageMesh struct {
	Name       string
	MeshConfig MeshConfig
	ConfigMap  *corev1.ConfigMap
	Deploy     *appsv1.Deployment
}

type MeshConfig struct {
	NodeId             string
	Canary             MeshCluster
	Target             MeshCluster
	TotalRoutingWeight uint32
}

// Binds together the Envoy cluster and the Kubernetes deployment.
type MeshCluster struct {
	Name          string
	RoutingWeight uint32
}

type MeshConfigSpec struct {
	CanaryDeployName string
	TargetDeployName string
	Opt              kconfig.Opt
}

type DeleteKageMeshSpec struct {
	CanaryDeployName string
	Opt              kconfig.Opt
}

type KageMeshSpec struct {
	Canary         *Canary
	LockdownTarget bool
	Opt            kconfig.Opt
}
