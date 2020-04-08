package model

import "github.com/kage-cloud/kage/core/kube/kconfig"

type KageSpec struct {
	Opt                     kconfig.Opt
	TargetDeployName        string
	CanaryRoutingPercentage uint32
}

type DeleteKageSpec struct {
	Opt        kconfig.Opt
	CanaryName string
}

type Kage struct {
	Mesh   KageMesh
	Canary Canary
}
