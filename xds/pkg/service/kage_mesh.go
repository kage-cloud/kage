package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kinformer"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"strings"
	"time"
)

const KageMeshServiceKey = "KageMeshService"

type KageMeshService interface {
	Fetch(canaryName string, opt kconfig.Opt) (*model.KageMesh, error)
	Create(spec *model.KageMeshSpec) (*model.KageMesh, error)
	CreateOrGet(canary *meta.Canary) (*meta.Xds, error)
	DeleteFromDeploy(kageMeshDeploy *appsv1.Deployment) error
	Delete(spec *model.DeleteKageMeshSpec) error
}

type kageMeshService struct {
	KubeClient                 kube.Client                `inject:"KubeClient"`
	KubeReaderService          KubeReaderService          `inject:"KubeReaderService"`
	KageMeshFactory            factory.KageMeshFactory    `inject:"KageMeshFactory"`
	XdsService                 XdsService                 `inject:"XdsService"`
	EndpointsControllerService EndpointsControllerService `inject:"EndpointsControllerService"`
	MeshConfigService          MeshConfigService          `inject:"MeshConfigService"`
	WatchService               WatchService               `inject:"WatchService"`
	LockdownService            LockdownService            `inject:"LockdownService"`
	EnvoyEndpointsService      EnvoyEndpointsService      `inject:"EnvoyEndpointsService"`
	InformerClient             kube.InformerClient        `inject:"InformerClient"`
}

func (k *kageMeshService) CreateOrGet(canary *meta.Canary) (*meta.Xds, error) {
	kageMeshName := k.genKageMeshName(canary)
	opt := kconfig.Opt{Namespace: canary.Source.Namespace}
	kageMeshDeploy, err := k.KubeReaderService.GetDeploy(kageMeshName, opt)
	if errors.IsNotFound(err) {

	}
}

func (k *kageMeshService) createKageMeshDeploy(name string, canary *meta.Canary, opt kconfig.Opt) (*appsv1.Deployment, *meta.XdsConfig, error) {
	xdsAnno := &meta.XdsConfig{
		NodeId: uuid.New().String(),
		Canary: meta.EnvoyConfig{
			ClusterName: "canary",
		},
		Source: meta.EnvoyConfig{
			ClusterName: "source",
		},
	}

	baseline, err := k.MeshConfigService.FromXdsConfig(xdsAnno)
	if err != nil {
		return nil, nil, err
	}

	cm := k.KageMeshFactory.BaselineConfigMap(name, baseline)

	dep := k.KageMeshFactory.Deploy(name)

	depObj, err := k.KubeClient.Create(dep, opt)
	if err != nil {
		return nil, nil, err
	}
}

func (k *kageMeshService) genKageMeshName(canary *meta.Canary) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-kage-mesh", canary.Source.Name, canary.Source.Kind))
}

func (k *kageMeshService) Deploy(ctx context.Context, source metav1.Object, xdsAnno *meta.Xds) error {
	opt := kconfig.Opt{Namespace: source.GetNamespace()}

	baseline, err := k.MeshConfigService.FromXdsConfig(xdsAnno)
	if err != nil {
		return err
	}

	cm := k.KageMeshFactory.BaselineConfigMap(xdsAnno.Config.MeshName, baseline)

	dep := k.KageMeshFactory.Deploy(xdsAnno.Config.MeshName)

	depObj, err := k.KubeClient.Create(dep, opt)
	if err != nil {
		return err
	}

	dep = depObj.(*appsv1.Deployment)

	informerSpec := kinformer.InformerSpec{
		NamespaceKind: ktypes.NamespaceKind{Namespace: source.GetNamespace(), Kind: ktypes.KindService},
		BatchDuration: 5 * time.Second,
		Filter: func(object metav1.Object) bool {
			baseObj := object.(runtime.Object)
			svcSelector := ktypes.GetLabelSelector(baseObj)
			if svcSelector == nil {
				return false
			}

			return svcSelector.Matches(labels.Set(source.GetLabels()))
		},
		Handlers: []kinformer.InformEventHandler{
			&kinformer.InformEventHandlerFuncs{
				OnWatch: nil,
				OnList: func(li metav1.ListInterface) error {
					svcs := li.(*corev1.ServiceList)

					for _, v := range svcs.Items {
						k.KubeReaderService.ListPods()
						if err := k.LockdownService.LockdownService2(&v, dep); err != nil {
							return err
						}

					}
				},
			},
		},
	}
	k.InformerClient.Inform(ctx)
}

func (k *kageMeshService) Delete(spec *model.DeleteKageMeshSpec) error {
	kageMeshDeploy, err := k.fetchDeployFromCanary(spec.CanaryDeployName, spec.Opt)
	if err != nil {
		return err
	}
	return k.DeleteFromDeploy(kageMeshDeploy)
}

func (k *kageMeshService) DeleteFromDeploy(kageMeshDeploy *appsv1.Deployment) error {
	meshConfig, err := k.MeshConfigService.Get(kageMeshDeploy)
	if err != nil {
		return err
	}

	targetName, err := canaryutil.TargetNameFromLabels(kageMeshDeploy.Labels)
	if err != nil {
		return err
	}

	canaryName, err := canaryutil.CanaryNameFromLabels(kageMeshDeploy.Labels)
	if err != nil {
		return err
	}

	blacklist := make([]appsv1.Deployment, 0)

	opt := kconfig.Opt{Namespace: kageMeshDeploy.Namespace}

	targetDeploy, err := k.KubeReaderService.GetDeploy(targetName, opt)
	if err == nil {
		blacklist = append(blacklist, *targetDeploy)
	}

	canaryDeploy, err := k.KubeReaderService.GetDeploy(canaryName, opt)
	if err == nil {
		blacklist = append(blacklist, *canaryDeploy)
	}

	return k.delete(meshConfig, blacklist, opt)
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

	baselineContent, err := k.MeshConfigService.BaselineConfig(meshConfig)
	if err != nil {
		return nil, err
	}

	meshConfigMap := k.KageMeshFactory.BaselineConfigMap(kageMeshName, spec.Canary.CanaryDeploy.Name, spec.Canary.TargetDeploy.Name, meshConfig, baselineContent)
	cm, err := k.KubeClient.UpsertConfigMap(meshConfigMap, opt)
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

	kageMeshDeploy.Labels = labels.Merge(kageMeshDeploy.Labels, spec.Canary.TargetDeploy.Labels)
	kageMeshDeploy.Spec.Template.Labels = labels.Merge(kageMeshDeploy.Spec.Template.Labels, spec.Canary.TargetDeploy.Spec.Template.Labels)

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

	cpSpec := &xds.ControlPlaneSpec{
		TargetDeploy: spec.Canary.TargetDeploy,
		CanaryDeploy: spec.Canary.CanaryDeploy,
		MeshConfig:   *meshConfig,
	}
	if err := k.XdsService.StartControlPlane(spec.Ctx, cpSpec); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		return nil, err
	}

	blacklist := []appsv1.Deployment{*spec.Canary.CanaryDeploy, *spec.Canary.TargetDeploy}
	if err := k.EndpointsControllerService.StartForDeploys(spec.Ctx, blacklist, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	err = k.WatchService.Deployment(spec.Ctx, kageMeshDeploy, 5*time.Second, &kinformer.InformEventHandlerFuncs{
		OnWatch: func(event watch.Event) error {
			if event.Type == watch.Deleted {
				log.WithField("node_id", meshConfig.NodeId).
					WithField("target", meshConfigSpec.TargetDeployName).
					WithField("canary", meshConfigSpec.CanaryDeployName).
					WithField("namespace", opt.Namespace).
					Debug("Stopping kage mesh")
				if err := k.delete(meshConfig, blacklist, opt); err != nil {
					log.WithField("node_id", meshConfig.NodeId).
						WithField("target", meshConfigSpec.TargetDeployName).
						WithField("canary", meshConfigSpec.CanaryDeployName).
						WithField("namespace", opt.Namespace).
						WithError(err).
						Error("Failed to stop kage mesh after it was deleted.")
					return err
				}
			}
			return nil
		},
	})

	if err != nil {
		return nil, err
	}

	return &model.KageMesh{
		Name:       kageMeshName,
		MeshConfig: *meshConfig,
		ConfigMap:  cm,
		Deploy:     kageMeshDeploy,
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

	cm, err := k.KubeReaderService.GetConfigMap(kageMeshDeploy.Name, opt)
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

	kageMeshDeploys, err := k.KubeReaderService.ListDeploys(selector, opt)
	if err != nil {
		return nil, err
	}

	if len(kageMeshDeploys) <= 0 {
		return nil, except.NewError("Canary %s could not be found.", except.ErrNotFound, canaryDeployName)
	}

	return &kageMeshDeploys[0], nil
}

func (k *kageMeshService) delete(meshConfig *model.MeshConfig, blacklist []appsv1.Deployment, opt kconfig.Opt) error {
	if err := k.EndpointsControllerService.Stop(blacklist, opt); err != nil {
		return err
	}

	if err := k.XdsService.StopControlPlane(meshConfig.NodeId); err != nil {
		return err
	}

	return nil
}
