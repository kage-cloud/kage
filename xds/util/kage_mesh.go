package util

import "fmt"

func GenKageMeshName(endpointsName string) string {
	return fmt.Sprintf("%s-kage-mesh", endpointsName)
}
