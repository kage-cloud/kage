package model

import (
	"github.com/eddieowens/kage/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
)

type KageMesh struct {
	Name                     string
	Deploy                   *appsv1.Deployment
	CanaryTrafficPercentage  uint32
	ServiceTrafficPercentage uint32
}

type KageMeshSpec struct {
	TargetDeployName        string
	CanaryTrafficPercentage int32
	Opt                     kconfig.Opt
}
