package model

import appsv1 "k8s.io/api/apps/v1"

const (
	TotalRoutingWeight = 100
)

type Canary struct {
	Name     string
	Version  string
	Deploy   *appsv1.Deployment
	KageMesh KageMesh
}
