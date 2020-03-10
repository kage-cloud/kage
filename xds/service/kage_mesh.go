package service

import (
	"fmt"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/model/consts"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/util"
	"github.com/eddieowens/kage/xds/watcher"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"math"
	"time"
)

const KageMeshServiceKey = "KageMeshService"

type KageMeshService interface {
	Fetch(endpointsName string, opt kconfig.Opt) (*model.KageMesh, error)
	Create(spec *model.KageMeshSpec) (*model.KageMesh, error)
}

type kageMeshService struct {
	KubeClient            kube.Client             `inject:"KubeClient"`
	StoreClient           snap.StoreClient        `inject:"StoreClient"`
	EnvoyStateService     EnvoyStateService       `inject:"EnvoyStateService"`
	KageMeshFactory       factory.KageMeshFactory `inject:"KageMeshFactory"`
	XdsWatcher            watcher.XdsWatcher      `inject:"XdsWatcher"`
	EndpointFinderService EndpointFinderService   `inject:"EndpointFinderService"`
}

func (k *kageMeshService) Create(spec *model.KageMeshSpec) (*model.KageMesh, error) {
	opt := spec.Opt
	target, err := k.KubeClient.GetDeploy(spec.TargetDeployName, opt)
	if err != nil {
		return nil, err
	}

	eps, err := k.EndpointFinderService.FindEndpoints(target, opt)
	if err != nil {
		return nil, err
	}

	baselineConfigMap := k.KageMeshFactory.BaselineConfigMap(spec.TargetDeployName, []byte(consts.BaselineConfig))
	if _, err := k.KubeClient.UpsertConfigMap(baselineConfigMap, opt); err != nil {
		return nil, err
	}

	kageMeshDeploy := k.KageMeshFactory.Deploy(
		spec.TargetDeployName,
		k.calcReplicasFromTraffic(target, spec.CanaryTrafficPercentage),
		k.findMatchingContainerPorts(k.aggContainerPorts(target), eps),
	)

	kageMeshDeploy, err = k.KubeClient.UpsertDeploy(kageMeshDeploy, opt)
	if err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		return nil, err
	}

	if err := k.KubeClient.WaitTillDeployReady(kageMeshDeploy.Name, time.Minute*1, opt); err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		return nil, err
	}

	endpointNames := make([]string, len(eps))
	for i, ep := range eps {
		endpointNames[i] = ep.Name
	}
	if err := k.XdsWatcher.WatchEndpoints(endpointNames); err != nil {
		return nil, err
	}

	lo := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", kageMeshDeploy.Name),
	}
	wi, err := k.KubeClient.InformDeploy(lo, opt)
	if err != nil {
		_ = k.KubeClient.DeleteConfigMap(baselineConfigMap.Name, opt)
		_ = k.KubeClient.DeleteDeploy(kageMeshDeploy.Name, opt)
		fmt.Println("Failed to watch the kage mesh deploy")
		return nil, err
	}

	go func() {
		for {
			r := <-wi.ResultChan()
			if r.Type == watch.Deleted {
				fmt.Println("Kage mesh", kageMeshDeploy.Name, "was deleted. Rolling back service", targetDeployName)

				break
			}
		}
	}()
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
					cps = append(cps, *util.ContainerPortFromEndpointPort(p))
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

func (k *kageMeshService) calcReplicasFromTraffic(target *appsv1.Deployment, trafficPercentage int32) int32 {
	reps := *target.Spec.Replicas
	percentReps := float32(reps) * (float32(trafficPercentage) / float32(model.TotalRoutingWeight))
	return int32(math.Ceil(float64(percentReps)))
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
