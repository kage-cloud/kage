package model

import "github.com/kage-cloud/kage/kube/kconfig"

type KageSpec struct {
	Opt                     kconfig.Opt
	TargetDeployName        string
	CanaryRoutingPercentage uint32
}

type DeleteKageSpec struct {
	Opt              kconfig.Opt
	TargetDeployName string
}
