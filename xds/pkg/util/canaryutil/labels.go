package canaryutil

import (
	"encoding/json"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	"github.com/kage-cloud/kage/xds/pkg/util"
)

func AppendCanaryLabels(targetDeployName string, labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}
	util.AppendKageLabels(labels)

	labels[consts.LabelKeyResource] = consts.LabelValueResourceCanary
	labels[consts.LabelKeyTarget] = targetDeployName
}

func GenCanaryLabels(targetDeployName string) map[string]string {
	m := map[string]string{}
	AppendCanaryLabels(targetDeployName, m)
	return m
}

func AppendKageMeshLabels(targetDeployName, canaryDeployName string, labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}
	util.AppendKageLabels(labels)
	labels[consts.LabelKeyTarget] = targetDeployName
	labels[consts.LabelKeyCanary] = canaryDeployName
	labels[consts.LabelKeyResource] = consts.LabelValueResourceKageMesh
}

func GenKageMeshLabels(targetDeployName, canaryDeployName string) map[string]string {
	m := map[string]string{}
	AppendKageMeshLabels(targetDeployName, canaryDeployName, m)
	return m
}

func GenKageMeshAnnotations(kageMeshMeta *model.MeshConfigAnnotation) map[string]string {
	b, _ := json.Marshal(kageMeshMeta)

	m := map[string]string{
		consts.AnnotationMeshConfig: string(b),
	}

	return m
}

func TargetNameFromLabels(l map[string]string) (string, error) {
	if v, ok := l[consts.LabelKeyTarget]; ok {
		return v, nil
	}
	return "", except.NewError("Missing the %s label", except.ErrInvalid, consts.LabelKeyTarget)
}

func CanaryNameFromLabels(l map[string]string) (string, error) {
	if v, ok := l[consts.LabelKeyCanary]; ok {
		return v, nil
	}
	return "", except.NewError("Missing the %s label", except.ErrInvalid, consts.LabelKeyCanary)
}
