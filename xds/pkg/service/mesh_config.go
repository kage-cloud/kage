package service

import (
	"bytes"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"text/template"
)

type MeshConfigService interface {
	Create() error
}

type meshConfigService struct {
	KageMeshFactory factory.KageMeshFactory `inject:"KageMeshFactory"`
	KubeClient      kube.Client             `inject:"KubeClient"`
}

func (m *meshConfigService) Create() error {
	tmpl := template.New("")
	t, err := tmpl.Parse(consts.BaselineConfig)

	buf := bytes.NewBuffer([]byte{})

	baseline := model.Baseline{
		NodeId:             "",
		NodeCluster:        "",
		XdsAddress:         "",
		XdsPort:            0,
		XdsClusterName:     "",
		AdminPort:          0,
		ServiceClusterName: "",
		CanaryClusterName:  "",
	}

	t.Execute(buf, "asd")
	baselineConfigMap := m.KageMeshFactory.BaselineConfigMap(kageMeshName, buf.Bytes())
	if _, err := m.KubeClient.UpsertConfigMap(baselineConfigMap, opt); err != nil {
		return nil, err
	}
}
