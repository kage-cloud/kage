package service

import (
	"github.com/eddieowens/kage/xds/exchange"
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/snap/store"
)

const TrafficControllerServiceKey = "TrafficControllerService"

type TrafficControllerService interface {
	Direct(req *exchange.DirectTrafficRequest) (*exchange.DirectTrafficResponse, error)
}

type trafficControllerService struct {
	StoreClient  snap.StoreClient     `inject:"StoreClient"`
	RouteFactory factory.RouteFactory `inject:"RouteFactory"`
}

func (t *trafficControllerService) Direct(req *exchange.DirectTrafficRequest) (*exchange.DirectTrafficResponse, error) {
	routes := t.RouteFactory.FromPercentage(req.EndpointName, req.Percentage)
	err := t.StoreClient.Set(&store.EnvoyState{
		Name:   req.EndpointName,
		Routes: routes,
	})
	if err != nil {
		return nil, err
	}

	return &exchange.DirectTrafficResponse{}, nil
}
