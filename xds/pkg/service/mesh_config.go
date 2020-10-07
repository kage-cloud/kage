package service

import (
	"bytes"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	"text/template"
)

const MeshConfigServiceKey = "MeshConfigService"

type MeshConfigService interface {
	Create(spec *model.MeshConfigSpec) (*model.MeshConfig, error)
	Get(kageMeshDeploy *appsv1.Deployment) (*model.MeshConfig, error)
	BaselineConfig(meshConfig *model.MeshConfig) ([]byte, error)
	FromXdsConfig(xdsAnno *meta.XdsConfig) ([]byte, error)
}

type meshConfigService struct {
	Config            *config.Config    `inject:"Config"`
	StoreClient       snap.StoreClient  `inject:"StoreClient"`
	EnvoyStateService EnvoyStateService `inject:"EnvoyStateService"`
}

func (m *meshConfigService) FromXdsConfig(xdsAnno *meta.XdsConfig) ([]byte, error) {
	tmpl := template.New("")

	t, err := tmpl.Parse(consts.BaselineConfig)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})

	baseline := model.Baseline{
		NodeId:             xdsAnno.NodeId,
		NodeCluster:        xdsAnno.Source.ClusterName,
		XdsAddress:         m.Config.Xds.Address,
		XdsPort:            m.Config.Xds.Port,
		AdminPort:          m.Config.Xds.AdminPort,
		ServiceClusterName: xdsAnno.Source.ClusterName,
		CanaryClusterName:  xdsAnno.Canary.ClusterName,
	}

	if err := t.Execute(buf, baseline); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *meshConfigService) BaselineConfig(meshConfig *model.MeshConfig) ([]byte, error) {
	tmpl := template.New("")

	t, err := tmpl.Parse(consts.BaselineConfig)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer([]byte{})

	baseline := model.Baseline{
		NodeId:             meshConfig.NodeId,
		NodeCluster:        meshConfig.Canary.Name,
		XdsAddress:         m.Config.Xds.Address,
		XdsPort:            m.Config.Xds.Port,
		AdminPort:          m.Config.Xds.AdminPort,
		ServiceClusterName: meshConfig.Target.Name,
		CanaryClusterName:  meshConfig.Canary.Name,
	}

	if err := t.Execute(buf, baseline); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
	nodeId, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	serviceName := spec.TargetDeployName
	canaryName := spec.CanaryDeployName

	meshConfig := &model.MeshConfig{
		NodeId: nodeId.String(),
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
