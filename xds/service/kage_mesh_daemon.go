package service

import (
	"github.com/eddieowens/kage/xds/model"
	appsv1 "k8s.io/api/apps/v1"
)

const KageMeshDaemonServiceKey = "KageMeshDaemonService"

type KageMeshDaemonService interface {
	Start(deployment *appsv1.Deployment) error
}

type kageMeshDaemonService struct {
	XdsEventHandler XdsEventHandler    `inject:"XdsEventHandler"`
	LockdownService LockdownService    `inject:"LockdownService"`
	WatchService    DeployWatchService `inject:"DeployWatchService"`
}

func (k *kageMeshDaemonService) Start(deployment *appsv1.Deployment) error {
	// Services must go first for the lockdown service to work properly.
	svcHandlers := make([]model.InformEventHandler, 1)
	svcHandlers[0] = k.LockdownService.DeployServicesEventHandler(deployment)

	podHandlers := make([]model.InformEventHandler, 2)
	podHandlers[0] = k.XdsEventHandler.DeployPodsEventHandler(deployment)
	podHandlers[1] = k.LockdownService.DeployPodsEventHandler(deployment)

	if err := k.WatchService.DeploymentPods(deployment, podHandlers...); err != nil {
		return err
	}

	if err := k.WatchService.DeploymentServices(deployment, svcHandlers...); err != nil {
		return err
	}

	return nil
}
