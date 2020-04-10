package model

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
)

type KageMeshAnnotation struct {
	NodeId            string
	CanaryClusterName string
	TargetClusterName string
}

type KageMesh struct {
	Name       string
	Deploy     *appsv1.Deployment
	MeshConfig MeshConfig
}

type MeshConfig struct {
	NodeId string
	Canary MeshDeployCluster
	Target MeshDeployCluster
}

// Binds together the Envoy cluster and the Kubernetes deployment.
type MeshDeployCluster struct {
	ClusterName       string
	DeployName        string
	TrafficPercentage uint32
}

type MeshConfigSpec struct {
	// The name for the config
	Name             string
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
