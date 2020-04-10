package snaputil

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

const (
	canarySuffix  = "kage-canary"
	serviceSuffix = "kage-service"
)

func GenTargetClusterName(name string) string {
	return fmt.Sprintf("%s-%s", name, serviceSuffix)
}

func GenCanaryClusterName(name string) string {
	return fmt.Sprintf("%s-%s", name, canarySuffix)
}

func IsCanaryName(name string) bool {
	return strings.HasSuffix(name, canarySuffix)
}

func NodeIdsFromConfigMap(configMap *corev1.ConfigMap) []string {
	nodeIds := make([]string, len(configMap.BinaryData))
	i := 0
	for k := range configMap.BinaryData {
		nodeIds[i] = k
		i++
	}
	return nodeIds
}
