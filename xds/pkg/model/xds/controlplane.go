package xds

import (
	"github.com/kage-cloud/kage/xds/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
)

type ControlPlaneSpec struct {
	TargetDeploy *appsv1.Deployment
	CanaryDeploy *appsv1.Deployment
	MeshConfig   model.MeshConfig
}
