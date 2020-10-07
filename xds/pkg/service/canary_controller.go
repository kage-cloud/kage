package service

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/exchange"
	"github.com/kage-cloud/kage/xds/pkg/model"
)

const CanaryControllerServiceKey = "CanaryControllerService"

type CanaryControllerService interface {
	Create(req *exchange.CreateCanaryRequest) (*exchange.CreateCanaryResponse, error)
	Delete(req *exchange.DeleteCanaryRequest) error
}

type canaryControllerService struct {
	KageService KageService    `inject:"KageService"`
	Config      *config.Config `inject:"Config"`
}

func (c *canaryControllerService) Create(req *exchange.CreateCanaryRequest) (*exchange.CreateCanaryResponse, error) {
	opt := kconfig.Opt{
		Namespace: req.Namespace,
	}

	spec := &model.KageSpec{
		Opt:                     opt,
		TargetControllerName:    req.Name,
		CanaryRoutingPercentage: req.CanaryRoutingPercentage,
	}

	kage, err := c.KageService.Create(spec)
	if err != nil {
		return nil, err
	}

	return &exchange.CreateCanaryResponse{
		Data: &exchange.Canary{
			Name:              kage.Canary.Name,
			TargetDeploy:      kage.Canary.TargetDeploy.Name,
			RoutingPercentage: kage.Canary.CanaryRoutingWeight,
		},
	}, nil
}

func (c *canaryControllerService) Delete(req *exchange.DeleteCanaryRequest) error {
	opt := kconfig.Opt{
		Namespace: req.Namespace,
	}

	deleteSpec := &model.DeleteKageSpec{
		Opt:        opt,
		CanaryName: req.Name,
	}

	return c.KageService.Delete(deleteSpec)
}
