package service

import (
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/xds/model"
)

const KageServiceKey = "KageService"

type KageService interface {
	Create(spec *model.KageSpec) (*model.Kage, error)
	Delete(spec *model.DeleteKageSpec) error
}

type kageService struct {
	KubeClient      kube.Client     `inject:"KubeClient"`
	KageMeshService KageMeshService `inject:"KageMeshService"`
	CanaryService   CanaryService   `inject:"CanaryService"`
}

func (k *kageService) Delete(spec *model.DeleteKageSpec) error {
	opt := spec.Opt
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

func (k *kageService) Create(spec *model.KageSpec) (*model.Kage, error) {
	opt := spec.Opt
	target, err := k.KubeClient.GetDeploy(spec.TargetDeployName, opt)
	if err != nil {
		return nil, err
	}

	canarySpec := &model.CreateCanarySpec{
		TargetDeploy:      target,
		TrafficPercentage: spec.CanaryRoutingPercentage,
		Opt:               opt,
	}

	canary, err := k.CanaryService.Create(canarySpec)
	if err != nil {
		return nil, err
	}

	kageMeshSpec := &model.KageMeshSpec{
		Canary:         canary,
		LockdownTarget: true,
		Opt:            opt,
	}

	kageMesh, err := k.KageMeshService.Create(kageMeshSpec)
	if err != nil {
		return nil, err
	}

	return &model.Kage{
		Mesh:   *kageMesh,
		Canary: *canary,
	}, nil
}
