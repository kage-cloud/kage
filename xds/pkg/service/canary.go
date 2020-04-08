package service

import (
	"fmt"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	"time"
)

const CanaryServiceKey = "CanaryService"

type CanaryService interface {
	Create(spec *model.CreateCanarySpec) (*model.Canary, error)
	Delete(spec *model.DeleteCanarySpec) error
}

type canaryService struct {
	KubeClient    kube.Client           `inject:"KubeClient"`
	CanaryFactory factory.CanaryFactory `inject:"CanaryFactory"`
}

func (c *canaryService) Delete(spec *model.DeleteCanarySpec) error {
	name := snaputil.GenCanaryName(spec.TargetDeploy.Name)
	if err := c.KubeClient.DeleteDeploy(name, spec.Opt); err != nil {
		return err
	}
	return nil
}

func (c *canaryService) Create(spec *model.CreateCanarySpec) (*model.Canary, error) {
	replicas := int32(1)
	if spec.TargetDeploy.Spec.Replicas != nil {
		replicas = *spec.TargetDeploy.Spec.Replicas
	}

	canaryReplicas := canaryutil.DeriveReplicaCountFromTraffic(replicas, spec.TrafficPercentage)

	name := snaputil.GenCanaryName(spec.TargetDeploy.Name)

	canary := c.CanaryFactory.FromDeployment(name, spec.TargetDeploy, canaryReplicas)

	dep, err := c.KubeClient.CreateDeploy(canary, spec.Opt)
	if err != nil {
		return nil, err
	}

	if err := c.KubeClient.WaitTillDeployReady(dep.Name, time.Second*30, spec.Opt); err != nil {
		if err = c.KubeClient.DeleteDeploy(dep.Name, spec.Opt); err != nil {
			fmt.Println("Failed to clean up canary", dep.Name, ":", err.Error())
		}
		return nil, err
	}

	return &model.Canary{
		Name:                name,
		TargetDeploy:        spec.TargetDeploy,
		CanaryDeploy:        dep,
		CanaryRoutingWeight: spec.TrafficPercentage,
		TotalRoutingWeight:  model.TotalRoutingWeight,
	}, nil
}
