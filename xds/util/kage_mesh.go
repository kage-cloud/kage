package util

import (
	"fmt"
	"github.com/eddieowens/kage/xds/model/consts"
)

func GenKageMeshName(endpointsName string) string {
	return fmt.Sprintf("%s-kage-mesh", endpointsName)
}

func AppendKageLabels(labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}

	labels[consts.LabelKeyDomain] = consts.Domain
}
