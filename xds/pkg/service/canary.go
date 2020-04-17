package service

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const CanaryServiceKey = "CanaryService"

type CanaryService interface {
	Create(spec *model.CreateCanarySpec) (*model.Canary, error)
	Delete(spec *model.DeleteCanarySpec) error
	Get(name string, opt kconfig.Opt) (*model.Canary, error)
}

type canaryService struct {
	KubeClient    kube.Client           `inject:"KubeClient"`
	CanaryFactory factory.CanaryFactory `inject:"CanaryFactory"`
}

func (c *canaryService) Get(name string, opt kconfig.Opt) (*model.Canary, error) {
	dep, err := c.KubeClient.Api().AppsV1().Deployments(opt.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	targetName, err := canaryutil.TargetNameFromLabels(dep.Labels)
	if err != nil {
		return nil, err
	}

	targetDeploy, err := c.KubeClient.Api().AppsV1().Deployments(opt.Namespace).Get(targetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &model.Canary{
		Name:                name,
		TargetDeploy:        targetDeploy,
		CanaryDeploy:        dep,
		CanaryRoutingWeight: 0,
		TotalRoutingWeight:  model.TotalRoutingWeight,
	}, nil
}

func (c *canaryService) Delete(spec *model.DeleteCanarySpec) error {
	if err := c.KubeClient.DeleteDeploy(spec.CanaryDeployName, spec.Opt); err != nil {
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

	name := snaputil.GenCanaryClusterName(spec.TargetDeploy.Name)

	canary := c.CanaryFactory.FromDeployment(name, spec.TargetDeploy, canaryReplicas)

	dep, err := c.KubeClient.CreateDeploy(canary, spec.Opt)
	if err != nil {
		return nil, err
	}

	if err := c.KubeClient.WaitTillDeployReady(dep.Name, time.Second*30, spec.Opt); err != nil {
		if err = c.KubeClient.DeleteDeploy(dep.Name, spec.Opt); err != nil {
			log.WithField("name", dep.Name).WithError(err).Error("Failed to clean up canary")
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
