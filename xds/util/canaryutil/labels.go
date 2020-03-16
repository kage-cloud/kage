package canaryutil

import (
	"github.com/eddieowens/kage/xds/model/consts"
	"github.com/eddieowens/kage/xds/util"
)

func AppendCanaryLabels(targetDeployName string, labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}
	util.AppendKageLabels(labels)

	labels[consts.LabelKeyResource] = consts.LabelValueResourceCanary
	labels[consts.LabelKeyFor] = targetDeployName
}

func GenCanaryLabels(targetDeployName string) map[string]string {
	m := map[string]string{}
	AppendCanaryLabels(targetDeployName, m)
	return m
}

func AppendKageMeshLabels(targetDeployName string, labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}
	util.AppendKageLabels(labels)
	labels[consts.LabelKeyFor] = targetDeployName
	labels[consts.LabelKeyResource] = consts.LabelValueResourceKageMesh
}

func GenKageMeshLabels(targetDeployName string) map[string]string {
	m := map[string]string{}
	AppendKageMeshLabels(targetDeployName, m)
	return m
}
