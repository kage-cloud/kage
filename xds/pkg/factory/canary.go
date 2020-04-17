package factory

import (
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const CanaryFactoryKey = "CanaryFactory"

type CanaryFactory interface {
	FromDeployment(name string, deployment *appsv1.Deployment, numReplicas int32) *appsv1.Deployment
}

type canaryFactory struct {
}

func (c *canaryFactory) FromDeployment(name string, deployment *appsv1.Deployment, numReplicas int32) *appsv1.Deployment {
	canary := deployment.DeepCopy()
	canary.Spec.Replicas = &numReplicas

	canary.Labels = labels.Merge(deployment.Labels, canaryutil.GenCanaryLabels(deployment.Name))
	canary.Name = name

	return canary
}
