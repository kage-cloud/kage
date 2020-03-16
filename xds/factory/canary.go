package factory

import (
	"github.com/eddieowens/kage/xds/util/canaryutil"
	appsv1 "k8s.io/api/apps/v1"
)

const CanaryFactoryKey = "CanaryFactory"

type CanaryFactory interface {
	FromDeployment(deployment *appsv1.Deployment) *appsv1.Deployment
}

type canaryFactory struct {
}

func (c *canaryFactory) FromDeployment(deployment *appsv1.Deployment, numReplicas int32) *appsv1.Deployment {
	canary := deployment.DeepCopy()
	canary.Spec.Replicas = &numReplicas

	canaryutil.AppendCanaryLabels(deployment.Name, deployment.Labels)

	return canary
}
