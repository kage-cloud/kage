package model

import (
	"context"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type MeshConfigAnnotation struct {
	// The metadata required for XDS indexed by the Node ID.
	XdsMetas map[string]XdsMeta `json:"xds_metas"`
}

type XdsMeta struct {
	NodeId      string `json:"node_id"`
	ClusterName string `json:"cluster_name"`
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
	Ctx            context.Context
	Canary         *Canary
	LockdownTarget bool
	Opt            kconfig.Opt
}
