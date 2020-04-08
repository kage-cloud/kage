package service

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	appsv1 "k8s.io/api/apps/v1"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	StartControlPlane(ctx context.Context, canary *model.Canary) error
	StopControlPlane(targetDeploy *appsv1.Deployment) error
	SetRoutingWeight(routingSpec *xds.RoutingSpec) error
}

type xdsService struct {
	XdsEventHandler XdsEventHandler      `inject:"XdsEventHandler"`
	WatchService    WatchService         `inject:"WatchService"`
	RouteFactory    factory.RouteFactory `inject:"RouteFactory"`
	StoreClient     snap.StoreClient     `inject:"StoreClient"`
}

func (x *xdsService) StopControlPlane(canary *model.Canary) error {
	if err := x.StoreClient.Delete(kubeutil.ObjectKey(canary.TargetDeploy)); err != nil {
		return err
	}

	if err := x.StoreClient.Delete(kubeutil.ObjectKey(canary.CanaryDeploy)); err != nil {
		return err
	}

	return nil
}

func (x *xdsService) SetRoutingWeight(routingSpec *xds.RoutingSpec) error {
	serviceName := snaputil.GenServiceName(routingSpec.TargetName)
	routes := x.RouteFactory.FromPercentage(routingSpec.CanaryName, serviceName, routingSpec.CanaryRoutingWeight, routingSpec.TotalRoutingWeight-routingSpec.CanaryRoutingWeight)

	state := &store.EnvoyState{
		NodeId: serviceName,
		Routes: routes,
	}

	return x.StoreClient.Set(state)
}

func (x *xdsService) StartControlPlane(ctx context.Context, canary *model.Canary) error {
	if x.controlPlaneExists(canary.TargetDeploy.Name) {
		return except.NewError("a canary for %s already exists", except.ErrAlreadyExists, canary.TargetDeploy.Name)
	}
	targetHandler := x.XdsEventHandler.DeployPodsEventHandler(canary.TargetDeploy)
	canaryHandler := x.XdsEventHandler.DeployPodsEventHandler(canary.CanaryDeploy)

	routingSpec := &xds.RoutingSpec{
		CanaryName:          canary.Name,
		TargetName:          canary.TargetDeploy.Name,
		CanaryRoutingWeight: canary.CanaryRoutingWeight,
		TotalRoutingWeight:  canary.TotalRoutingWeight,
	}
	if err := x.SetRoutingWeight(routingSpec); err != nil {
		return err
	}

	if err := x.WatchService.DeploymentPods(ctx, canary.TargetDeploy, 1, targetHandler); err != nil {
		return err
	}

	if err := x.WatchService.DeploymentPods(ctx, canary.CanaryDeploy, 1, canaryHandler); err != nil {
		return err
	}

	return nil
}

func (x *xdsService) controlPlaneExists(targetDeployName string) bool {
	_, err := x.StoreClient.Get(targetDeployName)
	return err != nil
}
