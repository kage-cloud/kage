package envoyepctlr

import (
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"k8s.io/apimachinery/pkg/labels"
)

type Spec struct {
	NodeId    string
	Selectors []labels.Selector
	Opt       kconfig.Opt
}
