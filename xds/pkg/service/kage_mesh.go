package service

import (
	"context"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/synchelpers"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/util"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"time"
)

const KageMeshServiceKey = "KageMeshService"

type KageMeshService interface {
	Fetch(canaryName string, opt kconfig.Opt) (*model.KageMesh, error)
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
	MeshConfigService          MeshConfigService          `inject:"MeshConfigService"`
	Map                        synchelpers.CancelFuncMap
}

func (k *kageMeshService) Delete(spec *model.DeleteKageMeshSpec) error {
	kageMeshDeploy, err := k.fetchDeployFromCanary(spec.CanaryDeployName, spec.Opt)
	if err != nil {
		return err
	}

	meshConfig, err := k.MeshConfigService.Get(kageMeshDeploy)
	if err != nil {
		return err
	}

	k.Map.Cancel(meshConfig.NodeId)

	if err := k.XdsService.StopControlPlane(meshConfig.NodeId); err != nil {
		return err
	}

	return nil
}

func (k *kageMeshService) Create(spec *model.KageMeshSpec) (*model.KageMesh, error) {
	opt := spec.Opt
	kageMeshName := util.GenKageMeshName(spec.Canary.Name)

	meshConfigSpec := &model.MeshConfigSpec{
		CanaryDeployName: spec.Canary.CanaryDeploy.Name,
		TargetDeployName: spec.Canary.TargetDeploy.Name,
		Opt:              opt,
	}

	meshConfig, err := k.MeshConfigService.Create(meshConfigSpec)
	if err != nil {
		return nil, err
	}

	containerPorts := make([]corev1.ContainerPort, 0)
	for _, cont := range spec.Canary.TargetDeploy.Spec.Template.Spec.Containers {
		for _, cp := range cont.Ports {
			containerPorts = append(containerPorts, cp)
		}
	}

	kageMeshDeploy := k.KageMeshFactory.Deploy(kageMeshName, spec.Canary.CanaryDeploy.Name, spec.Canary.TargetDeploy.Name, meshConfig, containerPorts)

	labels.Merge(kageMeshDeploy.Labels, spec.Canary.TargetDeploy.Labels)
	labels.Merge(kageMeshDeploy.Spec.Template.Labels, spec.Canary.TargetDeploy.Spec.Template.Labels)

	ctx, cancel := context.WithCancel(context.Background())
	k.Map.Add(meshConfig.NodeId, cancel)

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
				log.WithField("node_id", meshConfig.NodeId).
					WithField("target_deploy", meshConfigSpec.TargetDeployName).
					WithField("namespace", opt.Namespace).
					WithError(ctx.Err()).
					Debug("Stopping kage mesh")
				return
			}
		}
	}()

	cpSpec := &xds.ControlPlaneSpec{
		TargetDeploy: spec.Canary.TargetDeploy,
		CanaryDeploy: spec.Canary.CanaryDeploy,
		MeshConfig:   *meshConfig,
	}
	if err := k.XdsService.StartControlPlane(ctx, cpSpec); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		return nil, err
	}

	kageMeshDeploy, err = k.KubeClient.UpsertDeploy(kageMeshDeploy, opt)
	if err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		return nil, err
	}

	if err := k.KubeClient.WaitTillDeployReady(kageMeshDeploy.Name, time.Minute*1, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	blacklist, _ := metav1.LabelSelectorAsMap(spec.Canary.TargetDeploy.Spec.Selector)
	if err := k.EndpointsControllerService.StartWithBlacklistedEndpoints(ctx, blacklist, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	return &model.KageMesh{
		Name:       kageMeshName,
		Deploy:     kageMeshDeploy,
		MeshConfig: *meshConfig,
	}, nil
}

func (k *kageMeshService) Fetch(canaryDeployName string, opt kconfig.Opt) (*model.KageMesh, error) {
	kageMeshDeploy, err := k.fetchDeployFromCanary(canaryDeployName, opt)
	if err != nil {
		return nil, err
	}

	meshConfig, err := k.MeshConfigService.Get(kageMeshDeploy)
	if err != nil {
		return nil, err
	}

	cm, err := k.KubeClient.Api().CoreV1().ConfigMaps(opt.Namespace).Get(kageMeshDeploy.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &model.KageMesh{
		Name:       kageMeshDeploy.Name,
		MeshConfig: *meshConfig,
		ConfigMap:  cm,
		Deploy:     kageMeshDeploy,
	}, nil
}

func (k *kageMeshService) fetchDeployFromCanary(canaryDeployName string, opt kconfig.Opt) (*appsv1.Deployment, error) {
	selector := labels.SelectorFromSet(map[string]string{
		consts.LabelKeyCanary:   canaryDeployName,
		consts.LabelKeyResource: consts.LabelValueResourceKageMesh,
	})

	lo := metav1.ListOptions{
		LabelSelector: selector.String(),
	}

	kageMeshDeployLists, err := k.KubeClient.Api().AppsV1().Deployments(opt.Namespace).List(lo)
	if err != nil {
		return nil, err
	}

	if len(kageMeshDeployLists.Items) <= 0 {
		return nil, except.NewError("Canary %s has no mesh.", except.ErrNotFound, canaryDeployName)
	}

	return &kageMeshDeployLists.Items[0], nil
}

func kageMeshFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&kageMeshService{
		Map: synchelpers.NewCancelFuncMap(),
	})
}
