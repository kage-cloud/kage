package service

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	"github.com/kage-cloud/kage/xds/pkg/util"
	"github.com/kage-cloud/kage/xds/pkg/util/canaryutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"text/template"
)

const MeshConfigServiceKey = "MeshConfigService"

type MeshConfigService interface {
	CreateBaseline(spec *model.MeshConfigSpec) (*model.MeshConfig, error)
	FetchFromCanaryDeployName(name string, opt kconfig.Opt) (*model.MeshConfig, error)
}

type meshConfigService struct {
	KageMeshFactory   factory.KageMeshFactory `inject:"KageMeshFactory"`
	KubeClient        kube.Client             `inject:"KubeClient"`
	Config            *config.Config          `inject:"Config"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
	EnvoyStateService EnvoyStateService       `inject:"EnvoyStateService"`
}

func (m *meshConfigService) FetchFromCanaryDeployName(name string, opt kconfig.Opt) (*model.MeshConfig, error) {
	selector := labels.SelectorFromSet(map[string]string{
		consts.LabelKeyCanary:   name,
		consts.LabelKeyResource: consts.LabelValueResourceKageMesh,
	})

	lo := metav1.ListOptions{
		LabelSelector: selector.String(),
	}

	kageMeshDeployLists, err := m.KubeClient.Api().AppsV1().Deployments(opt.Namespace).List(lo)
	if err != nil {
		return nil, err
	}

	if len(kageMeshDeployLists.Items) <= 0 {
		return nil, except.NewError("Canary %s has no mesh.", except.ErrNotFound, name)
	}

	kageMeshDeploy := kageMeshDeployLists.Items[0]
	kageMeshMeta, err := util.MeshConfigAnnotation(kageMeshDeploy.Annotations)
	if err != nil {
		return nil, err
	}

	targetName, err := canaryutil.TargetNameFromLabels(kageMeshDeploy.Labels)
	if err != nil {
		return nil, err
	}

	state, err := m.StoreClient.Get(kageMeshMeta.NodeId)
	if err != nil {
		return nil, err
	}

	weight, err := m.EnvoyStateService.FetchCanaryRouteWeight(state)

	meshConfig := &model.MeshConfig{
		NodeId: kageMeshMeta.NodeId,
		Canary: model.MeshDeployCluster{
			ClusterName:       kageMeshMeta.CanaryClusterName,
			DeployName:        name,
			TrafficPercentage: weight,
		},
		Target: model.MeshDeployCluster{
			ClusterName:       kageMeshMeta.TargetClusterName,
			DeployName:        targetName,
			TrafficPercentage: model.TotalRoutingWeight - weight,
		},
	}

	return meshConfig, nil
}

func (m *meshConfigService) CreateBaseline(spec *model.MeshConfigSpec) (*model.MeshConfig, error) {
	tmpl := template.New("")
	t, err := tmpl.Parse(consts.BaselineConfig)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})

	nodeId, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	serviceName := snaputil.GenTargetClusterName(spec.TargetDeployName)
	canaryName := snaputil.GenCanaryClusterName(spec.CanaryDeployName)

	baseline := model.Baseline{
		NodeId:             nodeId.String(),
		NodeCluster:        spec.CanaryDeployName,
		XdsAddress:         m.Config.Xds.Address,
		XdsPort:            m.Config.Xds.Port,
		AdminPort:          m.Config.Xds.AdminPort,
		ServiceClusterName: serviceName,
		CanaryClusterName:  canaryName,
	}

	if err := t.Execute(buf, baseline); err != nil {
		return nil, err
	}

	meshConfig := &model.MeshConfig{
		NodeId: baseline.NodeId,
		Canary: model.MeshDeployCluster{
			ClusterName: canaryName,
			DeployName:  spec.CanaryDeployName,
		},
		Target: model.MeshDeployCluster{
			ClusterName: serviceName,
			DeployName:  spec.TargetDeployName,
		},
	}

	baselineConfigMap := m.KageMeshFactory.BaselineConfigMap(spec.Name, meshConfig, buf.Bytes())
	if _, err := m.KubeClient.UpsertConfigMap(baselineConfigMap, spec.Opt); err != nil {
		return nil, err
	}

	return meshConfig, nil
}
