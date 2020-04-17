package service

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	log "github.com/sirupsen/logrus"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	StartControlPlane(ctx context.Context, spec *xds.ControlPlaneSpec) error
	StopControlPlane(nodeId string) error
	SetRoutingWeight(meshConfig *model.MeshConfig) error
}

type xdsService struct {
	XdsEventHandler XdsEventHandler      `inject:"XdsEventHandler"`
	WatchService    WatchService         `inject:"WatchService"`
	RouteFactory    factory.RouteFactory `inject:"RouteFactory"`
	StoreClient     snap.StoreClient     `inject:"StoreClient"`
}

func (x *xdsService) StopControlPlane(nodeId string) error {
	if err := x.StoreClient.Delete(nodeId); err != nil {
		return err
	}

	return nil
}

func (x *xdsService) SetRoutingWeight(meshConfig *model.MeshConfig) error {
	routes := x.RouteFactory.FromPercentage(meshConfig)

	state := &store.EnvoyState{
		NodeId: meshConfig.NodeId,
		Routes: routes,
	}

	return x.StoreClient.Set(state)
}

func (x *xdsService) StartControlPlane(ctx context.Context, spec *xds.ControlPlaneSpec) error {
	if x.controlPlaneExists(spec.MeshConfig.NodeId) {
		return except.NewError("The control plane for canary %s already exists", except.ErrAlreadyExists, spec.CanaryDeploy.Name)
	}

	eventHandler := x.XdsEventHandler.DeployPodsEventHandler(spec.MeshConfig.NodeId)

	if err := x.SetRoutingWeight(&spec.MeshConfig); err != nil {
		return err
	}

	if err := x.WatchService.DeploymentPods(ctx, spec.TargetDeploy, 1, eventHandler); err != nil {
		return err
	}

	if err := x.WatchService.DeploymentPods(ctx, spec.CanaryDeploy, 1, eventHandler); err != nil {
		return err
	}

	log.WithField("name", spec.CanaryDeploy.Name).
		WithField("namespace", spec.CanaryDeploy.Namespace).
		WithField("node_id", spec.MeshConfig.NodeId).
		Debug("Added canary to control plane.")

	return nil
}

func (x *xdsService) controlPlaneExists(nodeId string) bool {
	_, err := x.StoreClient.Get(nodeId)
	exists := err == nil
	if !exists {
		log.WithField("node_id", nodeId).WithError(err).Debug("Control plane doesn't exist")
	}
	return exists
}
