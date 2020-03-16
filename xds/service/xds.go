package service

import (
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/snap/snaputil"
	"github.com/eddieowens/kage/xds/snap/store"
	appsv1 "k8s.io/api/apps/v1"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	InitializeControlPlane(deploy *appsv1.Deployment) (model.InformEventHandler, error)
}

type xdsService struct {
	XdsEventHandler XdsEventHandler      `inject:"XdsEventHandler"`
	RouteFactory    factory.RouteFactory `inject:"RouteFactory"`
	StoreClient     snap.StoreClient     `inject:"StoreClient"`
}

func (x *xdsService) InitializeControlPlane(deploy *appsv1.Deployment) (model.InformEventHandler, error) {
	handler := x.XdsEventHandler.EventHandler(deploy)

	canaryName := snaputil.GenCanaryName(deploy.Name)
	serviceName := snaputil.GenServiceName(deploy.Name)
	routes := x.RouteFactory.FromPercentage(canaryName, serviceName, 0, model.TotalRoutingWeight)

	if err := x.StoreClient.Set(&store.EnvoyState{Routes: routes}); err != nil {
		return nil, err
	}

	return handler, nil
}
