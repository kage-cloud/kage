package model

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/ktypes"
)

type KageSpec struct {
	TargetController        ktypes.ObjectRef
	CanaryRoutingPercentage uint32
}

type DeleteKageSpec struct {
	Opt        kconfig.Opt
	CanaryName string
}

type RegisterKageSpec struct {
	TargetController ktypes.ObjectRef
}

type Kage struct {
	Mesh   KageMesh
	Canary Canary
}
