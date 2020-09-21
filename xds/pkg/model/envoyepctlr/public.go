package envoyepctlr

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"k8s.io/apimachinery/pkg/labels"
)

type Spec struct {
	NodeId      string
	PodClusters []PodCluster
	Opt         kconfig.Opt
}

type PodCluster struct {
	Name     string
	Selector labels.Selector
}
