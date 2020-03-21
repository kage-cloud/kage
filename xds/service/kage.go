package service

import (
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/model"
	"os"
)

type KageService interface {
	Create(spec *model.KageSpec) error
	Delete(spec *model.DeleteKageSpec) error
}

type kageService struct {
	KubeClient      kube.Client     `inject:"KubeClient"`
	KageMeshService KageMeshService `inject:"KageMeshService"`
	CanaryService   CanaryService   `inject:"CanaryService"`
}

func (k *kageService) Delete(spec *model.DeleteKageSpec) error {
	opt := kconfig.Opt{
		Namespace: os.Getenv("NAMESPACE"),
	}
	targetDeploy, err := k.KubeClient.GetDeploy(spec.TargetDeployName, opt)
	if err != nil {
		return err
	}

	kageMeshSpec := &model.DeleteKageMeshSpec{
		TargetDeploy: targetDeploy,
		Opt:          opt,
	}

	if err := k.KageMeshService.Delete(kageMeshSpec); err != nil {
		return err
	}

	canarySpec := &model.DeleteCanarySpec{
		TargetDeploy: targetDeploy,
		Opt:          opt,
	}

	if err := k.CanaryService.Delete(canarySpec); err != nil {
		return err
	}

	return nil
}

func (k *kageService) Create(spec *model.KageSpec) error {
	opt := kconfig.Opt{
		Namespace: os.Getenv("NAMESPACE"),
	}
	target, err := k.KubeClient.GetDeploy(spec.TargetDeployName, opt)
	if err != nil {
		return err
	}

	canarySpec := &model.CanarySpec{
		TargetDeploy:      target,
		TrafficPercentage: spec.CanaryRoutingPercentage,
		Opt:               opt,
	}

	canary, err := k.CanaryService.Create(canarySpec)
	if err != nil {
		return err
	}

	kageMeshSpec := &model.KageMeshSpec{
		Canary:         canary,
		LockdownTarget: true,
		Opt:            opt,
	}

	_, err = k.KageMeshService.Create(kageMeshSpec)
	if err != nil {
		return err
	}

	return nil
}
