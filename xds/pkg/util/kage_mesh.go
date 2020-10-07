package util

import (
	"encoding/json"
	"fmt"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
)

func GenKageMeshName(targetDeployName string) string {
	return fmt.Sprintf("%s-kage-mesh", targetDeployName)
}

func AppendKageLabels(labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}

	labels[consts.LabelKeyDomain] = consts.Domain
}

func MeshConfigAnnotation(annotations map[string]string) (*model.MeshConfigAnnotation, error) {
	anno := new(model.MeshConfigAnnotation)

	if v, ok := annotations[consts.AnnotationMeshConfig]; ok {
		if err := json.Unmarshal([]byte(v), anno); err != nil {
			return nil, err
		}
		return anno, nil
	}
	return nil, except.NewError("Annotation %s could not be found", except.ErrNotFound, consts.AnnotationMeshConfig)
}

func IsKageMesh(l map[string]string) bool {
	if v, ok := l[consts.LabelKeyResource]; ok {
		return v == consts.LabelValueResourceKageMesh
	}
	return false
}
