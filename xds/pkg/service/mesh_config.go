package service

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/snaputil"
	"github.com/kage-cloud/kage/xds/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	"text/template"
)

const MeshConfigServiceKey = "MeshConfigService"

type MeshConfigService interface {
	Create(spec *model.MeshConfigSpec) (*model.MeshConfig, error)
	Get(kageMeshDeploy *appsv1.Deployment) (*model.MeshConfig, error)
}

type meshConfigService struct {
	Config            *config.Config    `inject:"Config"`
	StoreClient       snap.StoreClient  `inject:"StoreClient"`
	EnvoyStateService EnvoyStateService `inject:"EnvoyStateService"`
}

func (m *meshConfigService) Get(kageMeshDeploy *appsv1.Deployment) (*model.MeshConfig, error) {
	kageMeshMeta, err := util.MeshConfigAnnotation(kageMeshDeploy.Annotations)
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
		Canary: model.MeshCluster{
			Name:          kageMeshMeta.CanaryClusterName,
			RoutingWeight: weight,
		},
		Target: model.MeshCluster{
			Name:          kageMeshMeta.TargetClusterName,
			RoutingWeight: model.TotalRoutingWeight - weight,
		},
		TotalRoutingWeight: model.TotalRoutingWeight,
	}

	return meshConfig, nil
}

func (m *meshConfigService) Create(spec *model.MeshConfigSpec) (*model.MeshConfig, error) {
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
		Canary: model.MeshCluster{
			Name: canaryName,
		},
		Target: model.MeshCluster{
			Name:          serviceName,
			RoutingWeight: model.TotalRoutingWeight,
		},
		TotalRoutingWeight: model.TotalRoutingWeight,
	}

	return meshConfig, nil
}
