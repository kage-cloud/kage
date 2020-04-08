package kubeutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Finds a port with the specified name. If a matching ContainerPort name is found, the container is returned as well as
// the index of the ContainerPort. If the name is not found, default values are returned.
func FindPortByTargetPort(is intstr.IntOrString, ports []corev1.ContainerPort) (conPort *corev1.ContainerPort, idx int) {
	for i, cp := range ports {
		if is.Type == intstr.String && cp.Name == is.StrVal || cp.ContainerPort == is.IntVal {
			conPort, idx = &cp, i
		}
	}
	return
}
