package service

import (
	"fmt"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/synchelpers"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/model"
	"github.com/kage-cloud/kage/xds/model/consts"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/util"
	"github.com/kage-cloud/kage/xds/util/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
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
	KubeClient            kube.Client             `inject:"KubeClient"`
	StoreClient           snap.StoreClient        `inject:"StoreClient"`
	EnvoyStateService     EnvoyStateService       `inject:"EnvoyStateService"`
	KageMeshFactory       factory.KageMeshFactory `inject:"KageMeshFactory"`
	XdsService            XdsService              `inject:"XdsService"`
	LockdownService       LockdownService         `inject:"LockdownService"`
	StopperHandlerService StopperHandlerService   `inject:"StopperHandlerService"`
}

func (k *kageMeshService) Delete(spec *model.DeleteKageMeshSpec) error {
	objKey := kubeutil.ObjectKey(spec.TargetDeploy)

	k.StopperHandlerService.Stop(objKey, nil)

	kageMeshName := util.GenKageMeshName(spec.TargetDeploy.Name)

	if err := k.KubeClient.DeleteDeploy(kageMeshName, spec.Opt); err != nil {
		fmt.Println("failed to delete kage mesh deploy", kageMeshName, "in", spec.Opt.Namespace)
	}

	if err := k.KubeClient.DeleteConfigMap(kageMeshName, spec.Opt); err != nil {
		fmt.Println("failed to delete kage mesh configmap", kageMeshName, "in", spec.Opt.Namespace)
	}

	if err := k.XdsService.StopControlPlane(objKey); err != nil {
		return err
	}

	if k.LockdownService.IsLockedDown(spec.TargetDeploy) {
		if err := k.LockdownService.ReleaseDeploy(spec.TargetDeploy); err != nil {
			return err
		}
	}

	return nil
}

func (k *kageMeshService) Create(spec *model.KageMeshSpec) (*model.KageMesh, error) {
	opt := spec.Opt
	if err := k.XdsService.InitializeControlPlane(spec.Canary); err != nil {
		return nil, err
	}

	kageMeshName := util.GenKageMeshName(spec.Canary.TargetDeploy.Name)
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

	// Possibly kill the canary if the target deployment is deleted?
	wi := k.KubeClient.InformDeploy(func(object metav1.Object) bool {
		return object.GetName() == kageMeshName
	})

	objKey := kubeutil.ObjectKey(kageMeshDeploy)
	stopper, errChan := synchelpers.NewErrChanStopper(func(err error) {
		k.StopperHandlerService.Remove(objKey)
	})
	k.StopperHandlerService.Add(objKey, stopper)

	go func() {
		for {
			select {
			case r := <-wi:
				if r.Type == watch.Deleted {
					// TODO: something after the kage mesh deploy is deleted.
					//fmt.Println("Kage mesh", kageMeshDeployName, "was deleted. Rolling back service", targetDeployName)
					break
				}

			case err := <-errChan:
				if err != nil {
					fmt.Println("stopping kage mesh", kageMeshName, ":", err.Error())
					return
				}
			}
		}
	}()

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

	if spec.LockdownTarget {
		if err := k.LockdownService.LockdownDeploy(spec.Canary.TargetDeploy); err != nil {
			return nil, err
		}
	}

	return &model.KageMesh{
		Name:                     kageMeshName,
		Deploy:                   kageMeshDeploy,
		CanaryTrafficPercentage:  spec.Canary.TrafficPercentage,
		ServiceTrafficPercentage: model.TotalRoutingWeight,
	}, nil
}

// Extract all ContainerPorts from Endpoints in eps which will route to ContainerPorts in toMatch.
func (k *kageMeshService) findMatchingContainerPorts(toMatch []corev1.ContainerPort, eps []corev1.Endpoints) []corev1.ContainerPort {
	toMatchByPort := map[int32][]corev1.ContainerPort{}
	for _, tm := range toMatch {
		v, ok := toMatchByPort[tm.ContainerPort]
		if ok {
			v = append(v, tm)
		} else {
			v = []corev1.ContainerPort{tm}
		}
	}
	cps := make([]corev1.ContainerPort, 0)
	for _, ep := range eps {
		for _, ss := range ep.Subsets {
			for _, p := range ss.Ports {
				if _, ok := toMatchByPort[p.Port]; ok {
					cps = append(cps, *util.ContainerPortFromEndpointPort(&p))
				}
			}
		}
	}

	return cps
}

func (k *kageMeshService) aggContainerPorts(dep *appsv1.Deployment) []corev1.ContainerPort {
	cps := make([]corev1.ContainerPort, 0)
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, cp := range c.Ports {
			cps = append(cps, cp)
		}
	}
	return cps
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
