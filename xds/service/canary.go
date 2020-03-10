package service

import (
	"fmt"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/snap"
)

const CanaryServiceKey = "CanaryService"

type CanaryService interface {
	GenCanaryName(name, version string) string
	Fetch(name string, opt kconfig.Opt) (*model.Canary, error)
	IsCanaried(name string, opt kconfig.Opt) bool
}

type canaryService struct {
	KubeClient      kube.Client      `inject:"KubeClient"`
	StoreClient     snap.StoreClient `inject:"StoreClient"`
	KageMeshService KageMeshService  `inject:"KageMeshService"`
}

func (c *canaryService) IsCanaried(name string, opt kconfig.Opt) bool {
	c.KubeClient.GetDeploy()
}

func (c *canaryService) GenCanaryName(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}

func (c *canaryService) Fetch(name string, opt kconfig.Opt) (*model.Canary, error) {
	dep, err := c.KubeClient.GetDeploy(name, opt)
	if err != nil {
		return nil, err
	}
}
