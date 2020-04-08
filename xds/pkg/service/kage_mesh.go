package service

import (
	"context"
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/core/synchelpers"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const KageMeshServiceKey = "KageMeshService"

type KageMeshService interface {
	Fetch(endpointsName string, opt kconfig.Opt) (*model.KageMesh, error)
	Create(spec *model.KageMeshSpec) (*model.KageMesh, error)
	Delete(spec *model.DeleteKageMeshSpec) error
}

type kageMeshService struct {
	KubeClient                 kube.Client                `inject:"KubeClient"`
	StoreClient                snap.StoreClient           `inject:"StoreClient"`
	EnvoyStateService          EnvoyStateService          `inject:"EnvoyStateService"`
	KageMeshFactory            factory.KageMeshFactory    `inject:"KageMeshFactory"`
	XdsService                 XdsService                 `inject:"XdsService"`
	LockdownService            LockdownService            `inject:"LockdownService"`
	EndpointsControllerService EndpointsControllerService `inject:"EndpointsControllerService"`
	Map                        synchelpers.CancelFuncMap
}

func (k *kageMeshService) Delete(spec *model.DeleteKageMeshSpec) error {
	kageMeshName := util.GenKageMeshName(spec.Canary.TargetDeploy.Name)

	kageMeshDep, err := k.KubeClient.GetDeploy(kageMeshName, spec.Opt)
	if err != nil {
		return err
	}

	k.Map.Cancel(kubeutil.ObjectKey(kageMeshDep))

	if err := k.KubeClient.DeleteDeploy(kageMeshName, spec.Opt); err != nil {
		log.WithError(err).
			WithField("namespace", spec.Opt.Namespace).
			WithField("name", kageMeshName).
			Error("Failed to delete kage mesh deploy")
	}

	if err := k.KubeClient.DeleteConfigMap(kageMeshName, spec.Opt); err != nil {
		log.WithError(err).
			WithField("namespace", spec.Opt.Namespace).
			WithField("name", kageMeshName).
			Error("Failed to delete kage mesh configmap")
	}

	if err := k.XdsService.StopControlPlane(spec.Canary); err != nil {
		return err
	}

	return nil
}

func (k *kageMeshService) Create(spec *model.KageMeshSpec) (*model.KageMesh, error) {
	opt := spec.Opt

	kageMeshName := util.GenKageMeshName(spec.Canary.Name)
	baselineConfigMap := k.KageMeshFactory.BaselineConfigMap(kageMeshName, []byte(consts.BaselineConfig))
	if _, err := k.KubeClient.UpsertConfigMap(baselineConfigMap, opt); err != nil {
		return nil, err
	}

	containerPorts := make([]corev1.ContainerPort, 0)
	for _, cont := range spec.Canary.TargetDeploy.Spec.Template.Spec.Containers {
		for _, cp := range cont.Ports {
			containerPorts = append(containerPorts, cp)
		}
	}
	kageMeshDeploy := k.KageMeshFactory.Deploy(kageMeshName, spec.Canary.TargetDeploy.Name, containerPorts, spec.Canary.TargetDeploy.Labels, spec.Canary.TargetDeploy.Spec.Template.Labels)

	objKey := kubeutil.ObjectKey(kageMeshDeploy)
	ctx, cancel := context.WithCancel(context.Background())
	k.Map.Add(objKey, cancel)

	// Possibly kill the canary if the target deployment is deleted?
	wi := k.KubeClient.InformDeploy(func(object metav1.Object) bool {
		return object.GetName() == kageMeshName
	})

	go func() {
		for {
			select {
			case r := <-wi:
				if r.Type == watch.Deleted {
					// TODO: something after the kage mesh deploy is deleted.
					//fmt.Println("Kage mesh", kageMeshDeployName, "was deleted. Rolling back service", targetDeployName)
					break
				}

			case <-ctx.Done():
				if ctx.Err() != nil {
					fmt.Println("stopping kage mesh", kageMeshName, ":", ctx.Err().Error())
				}
				return
			}
		}
	}()

	if err := k.XdsService.StartControlPlane(ctx, spec.Canary); err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		return nil, err
	}

	kageMeshDeploy, err := k.KubeClient.UpsertDeploy(kageMeshDeploy, opt)
	if err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		return nil, err
	}

	if err := k.KubeClient.WaitTillDeployReady(kageMeshDeploy.Name, time.Minute*1, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	blacklist, _ := metav1.LabelSelectorAsMap(spec.Canary.TargetDeploy.Spec.Selector)
	if err := k.EndpointsControllerService.StartWithBlacklistedEndpoints(ctx, blacklist, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	return &model.KageMesh{
		Name:                     kageMeshName,
		Deploy:                   kageMeshDeploy,
		CanaryTrafficPercentage:  spec.Canary.CanaryRoutingWeight,
		ServiceTrafficPercentage: model.TotalRoutingWeight,
	}, nil
}

func (k *kageMeshService) Fetch(endpointsName string, opt kconfig.Opt) (*model.KageMesh, error) {
	dep, err := k.KubeClient.GetDeploy(util.GenKageMeshName(endpointsName), opt)
	if err != nil {
		return nil, err
	}

	state, err := k.StoreClient.Get(endpointsName)
	if err != nil {
		return nil, err
	}

	weight, err := k.EnvoyStateService.FetchCanaryRouteWeight(state)
	if err != nil {
		return nil, err
	}

	return &model.KageMesh{
		Name:                     dep.Name,
		Deploy:                   dep,
		CanaryTrafficPercentage:  weight,
		ServiceTrafficPercentage: model.TotalRoutingWeight - weight,
	}, nil
}

func kageMeshFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&kageMeshService{
		Map: synchelpers.NewCancelFuncMap(),
	})
}
