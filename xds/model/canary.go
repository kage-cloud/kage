package model

import (
	"github.com/kage-cloud/kage/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	TotalRoutingWeight = 100
)

type CanarySpec struct {
	TargetDeploy      *appsv1.Deployment
	TrafficPercentage uint32
	Opt               kconfig.Opt
}

type DeleteCanarySpec struct {
	TargetDeploy *appsv1.Deployment
	Opt          kconfig.Opt
}

type Canary struct {
	Name                string
	TargetDeploy        *appsv1.Deployment
	CanaryDeploy        *appsv1.Deployment
	CanaryRoutingWeight uint32
	TotalRoutingWeight  uint32
}
