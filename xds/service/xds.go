package service

import (
	"github.com/kage-cloud/kage/xds/except"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/model"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/snap/snaputil"
	"github.com/kage-cloud/kage/xds/snap/store"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	InitializeControlPlane(canary *model.Canary) error
	StopControlPlane(targetDeployKey string) error
}

type xdsService struct {
	XdsEventHandler       XdsEventHandler       `inject:"XdsEventHandler"`
	DeployWatchService    DeployWatchService    `inject:"DeployWatchService"`
	RouteFactory          factory.RouteFactory  `inject:"RouteFactory"`
	StoreClient           snap.StoreClient      `inject:"StoreClient"`
	StopperHandlerService StopperHandlerService `inject:"StopperHandlerService"`
}

func (x *xdsService) StopControlPlane(targetDeployKey string) error {
	if !x.StopperHandlerService.Exists(targetDeployKey) {
		return except.NewError("%s could not be found", except.ErrNotFound, targetDeployKey)
	}
	x.StopperHandlerService.Stop(targetDeployKey, nil)
	return nil
}

func (x *xdsService) InitializeControlPlane(canary *model.Canary) error {
	if x.controlPlaneExists(canary.TargetDeploy.Name) {
		return except.NewError("a canary for %s already exists", except.ErrAlreadyExists, canary.TargetDeploy.Name)
	}
	handler := x.XdsEventHandler.DeployPodsEventHandler(canary.TargetDeploy)

	serviceName := snaputil.GenServiceName(canary.TargetDeploy.Name)
	routes := x.RouteFactory.FromPercentage(canary.Name, serviceName, canary.TrafficPercentage, model.TotalRoutingWeight)

	if err := x.StoreClient.Set(&store.EnvoyState{Routes: routes}); err != nil {
		return err
	}

	err := x.DeployWatchService.DeploymentPods(canary.TargetDeploy, handler)
	if err != nil {
		return err
	}

	return nil
}

func (x *xdsService) controlPlaneExists(targetDeployName string) bool {
	return x.StopperHandlerService.Exists(targetDeployName)
}
