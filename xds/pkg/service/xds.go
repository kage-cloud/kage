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
	log "github.com/sirupsen/logrus"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	StartControlPlane(ctx context.Context, canary *model.Canary) error
	StopControlPlane(canary *model.Canary) error
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
		NodeId: routingSpec.CanaryName,
		Routes: routes,
	}

	return x.StoreClient.Set(state)
}

func (x *xdsService) StartControlPlane(ctx context.Context, canary *model.Canary) error {
	if x.controlPlaneExists(kubeutil.ObjectKey(canary.CanaryDeploy)) {
		return except.NewError("a canary called %s already exists", except.ErrAlreadyExists, canary.Name)
	}

	if x.controlPlaneExists(kubeutil.ObjectKey(canary.TargetDeploy)) {
		log.WithField("name", canary.TargetDeploy.Name).WithField("namespace", canary.TargetDeploy.Namespace).Debug("Already managed by control plane.")
	} else {
		targetHandler := x.XdsEventHandler.DeployPodsEventHandler(canary.TargetDeploy)
		if err := x.WatchService.DeploymentPods(ctx, canary.TargetDeploy, 1, targetHandler); err != nil {
			return err
		}
	}
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

	if err := x.WatchService.DeploymentPods(ctx, canary.CanaryDeploy, 1, canaryHandler); err != nil {
		return err
	}

	log.WithField("name", canary.CanaryDeploy.Name).WithField("namespace", canary.CanaryDeploy.Name).Debug("Added canary to control plane.")

	return nil
}

func (x *xdsService) controlPlaneExists(key string) bool {
	_, err := x.StoreClient.Get(key)
	return err != nil
}
