package service

import (
	"context"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/synchelpers"
	"github.com/kage-cloud/kage/xds/pkg/model"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const KageServiceKey = "KageService"

type KageService interface {
	Create(spec *model.KageSpec) (*model.Kage, error)
	Register()
	Delete(spec *model.DeleteKageSpec) error
}

type kageService struct {
	KubeReaderService KubeReaderService `inject:"KubeReaderService"`
	KageMeshService   KageMeshService   `inject:"KageMeshService"`
	CanaryService     CanaryService     `inject:"CanaryService"`
	WatchService      WatchService      `inject:"WatchService"`
	Map               synchelpers.CancelFuncMap
}

func (k *kageService) Delete(spec *model.DeleteKageSpec) error {
	opt := spec.Opt

	k.Map.Cancel(spec.CanaryName)

	kageMeshSpec := &model.DeleteKageMeshSpec{
		CanaryDeployName: spec.CanaryName,
		Opt:              opt,
	}
	if err := k.KageMeshService.Delete(kageMeshSpec); err != nil {
		return err
	}

	canarySpec := &model.DeleteCanarySpec{
		CanaryDeployName: spec.CanaryName,
		Opt:              opt,
	}

	if err := k.CanaryService.Delete(canarySpec); err != nil {
		return err
	}

	return nil
}

func (k *kageService) Create(spec *model.KageSpec) (*model.Kage, error) {
	opt := kconfig.Opt{Namespace: spec.TargetController.Namespace}
	target, err := k.KubeReaderService.Get(spec.TargetController.Name, spec.TargetController.Kind, opt)
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

	ctx, cancel := context.WithCancel(context.Background())

	kageMeshSpec := &model.KageMeshSpec{
		Ctx:            ctx,
		Canary:         canary,
		LockdownTarget: true,
		Opt:            opt,
	}

	k.Map.Add(canary.Name, cancel)

	kageMesh, err := k.KageMeshService.Create(kageMeshSpec)
	if err != nil {
		return nil, err
	}

	err = k.WatchService.Deployment(ctx, canary.CanaryDeploy, 5*time.Second, &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			switch event.Type {
			case watch.Deleted:
				log.WithField("name", canary.CanaryDeploy.Name).
					WithField("namespace", opt.Namespace).
					Info("Canary deploy deleted")
				if err := k.KageMeshService.DeleteFromDeploy(kageMesh.Deploy); err != nil {
					log.WithField("name", kageMesh.Name).
						WithField("namespace", opt.Namespace).
						WithField("canary", canary.Name).
						WithError(err).
						Error("Failed to clean up mesh after canary was deleted.")
				}
				log.WithField("name", kageMesh.Name).
					WithField("namespace", opt.Namespace).
					WithField("canary", canary.Name).
					Info("Successfully deleted mesh after canary was deleted.")
				return err
			}
			return nil
		},
	})
	if err != nil {
		log.WithField("name", kageMesh.Name).
			WithField("namespace", opt.Namespace).
			WithField("canary", canary.Name).
			WithError(err).
			Error("Failed to watch the canary. Cleaning up.")

		spec := &model.DeleteKageSpec{
			Opt:        opt,
			CanaryName: canary.CanaryDeploy.Name,
		}

		if err := k.Delete(spec); err != nil {
			log.WithField("name", kageMesh.Name).
				WithField("namespace", opt.Namespace).
				WithField("canary", canary.Name).
				WithError(err).
				Error("Failed to clean up")
		}
		return nil, err
	}

	return &model.Kage{
		Mesh:   *kageMesh,
		Canary: *canary,
	}, nil
}

func kageServiceFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&kageService{
		Map: synchelpers.NewCancelFuncMap(),
	})
}
