package service

import (
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
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
	DeleteFromDeploy(kageMeshDeploy *appsv1.Deployment) error
	Delete(spec *model.DeleteKageMeshSpec) error
}

type kageMeshService struct {
	KubeClient                 kube.Client                `inject:"KubeClient"`
	KageMeshFactory            factory.KageMeshFactory    `inject:"KageMeshFactory"`
	XdsService                 XdsService                 `inject:"XdsService"`
	EndpointsControllerService EndpointsControllerService `inject:"EndpointsControllerService"`
	MeshConfigService          MeshConfigService          `inject:"MeshConfigService"`
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

	targetDeploy, err := k.KubeClient.Api().AppsV1().Deployments(kageMeshDeploy.Namespace).Get(targetName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	blacklist, _ := metav1.LabelSelectorAsMap(targetDeploy.Spec.Selector)

	return k.delete(meshConfig, blacklist, kconfig.Opt{Namespace: kageMeshDeploy.Namespace})
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

	cpSpec := &xds.ControlPlaneSpec{
		TargetDeploy: spec.Canary.TargetDeploy,
		CanaryDeploy: spec.Canary.CanaryDeploy,
		MeshConfig:   *meshConfig,
	}
	if err := k.XdsService.StartControlPlane(spec.Ctx, cpSpec); err != nil {
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
	if err := k.EndpointsControllerService.StartWithBlacklistedEndpoints(spec.Ctx, blacklist, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(kageMeshName, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	wi := k.KubeClient.InformDeploy(func(object metav1.Object) bool {
		return object.GetName() == kageMeshName
	})

	go func() {
		for {
			select {
			case r := <-wi:
				if r.Type == watch.Deleted {
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
					} else {
						return
					}
					break
				}

			case <-spec.Ctx.Done():
				log.WithField("node_id", meshConfig.NodeId).
					WithField("target", meshConfigSpec.TargetDeployName).
					WithField("canary", meshConfigSpec.CanaryDeployName).
					WithField("namespace", opt.Namespace).
					WithError(spec.Ctx.Err()).
					Debug("Stopping kage mesh")
				return
			}
		}
	}()

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

func (k *kageMeshService) delete(meshConfig *model.MeshConfig, blacklist labels.Set, opt kconfig.Opt) error {
	if err := k.EndpointsControllerService.Stop(blacklist, opt); err != nil {
		return err
	}

	if err := k.XdsService.StopControlPlane(meshConfig.NodeId); err != nil {
		return err
	}

	return nil
}
