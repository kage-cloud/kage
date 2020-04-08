package model

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
)

type KageMesh struct {
	Name                     string
	Deploy                   *appsv1.Deployment
	CanaryTrafficPercentage  uint32
	ServiceTrafficPercentage uint32
}

type DeleteKageMeshSpec struct {
	TargetDeploy *appsv1.Deployment
	Opt          kconfig.Opt
}

type KageMeshSpec struct {
	Canary         *Canary
	LockdownTarget bool
	Opt            kconfig.Opt
}
