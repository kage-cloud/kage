package service

import (
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/xds/factory"
	appsv1 "k8s.io/api/apps/v1"
)

const CanaryServiceKey = "CanaryService"

type CanaryService interface {
	Canary(deployment *appsv1.Deployment) (*appsv1.Deployment, error)
}

type canaryService struct {
	KubeClient    kube.Client           `inject:"KubeClient"`
	CanaryFactory factory.CanaryFactory `inject:"CanaryFactory"`
}

func (c *canaryService) Canary(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {

}
