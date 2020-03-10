package snaputil

import (
	"fmt"
	"strings"
)

const (
	canarySuffix  = "kage-canary"
	serviceSuffix = "kage-service"
)

func GenServiceName(endpointsName string) string {
	return fmt.Sprintf("%s-%s", endpointsName, serviceSuffix)
}

func GenCanaryName(endpointsName string) string {
	return fmt.Sprintf("%s-%s", endpointsName, canarySuffix)
}

func IsCanaryName(name string) bool {
	return strings.HasSuffix(name, canarySuffix)
}
