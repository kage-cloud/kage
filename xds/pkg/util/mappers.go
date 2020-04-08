package util

import (
	envcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/kage-cloud/kage/core/except"
	corev1 "k8s.io/api/core/v1"
)

func KubeProtocolToSocketAddressProtocol(protocol corev1.Protocol) (envcore.SocketAddress_Protocol, error) {
	switch protocol {
	case corev1.ProtocolSCTP:
		return -1, except.NewError("SCTP is not a supported protocol", except.ErrUnsupported)
	case corev1.ProtocolTCP:
		return envcore.SocketAddress_TCP, nil
	case corev1.ProtocolUDP:
		return envcore.SocketAddress_UDP, nil
	}
	return -1, except.NewError("Unknown protocol", except.ErrUnsupported)
}

func ContainerPortsFromEndpoints(endpoints *corev1.Endpoints) []corev1.ContainerPort {
	containerPorts := make([]corev1.ContainerPort, 0)
	for _, ss := range endpoints.Subsets {
		for _, port := range ss.Ports {
			containerPorts = append(containerPorts, *ContainerPortFromEndpointPort(&port))
		}
	}
	return containerPorts
}

func ContainerPortFromEndpointPort(port *corev1.EndpointPort) *corev1.ContainerPort {
	return &corev1.ContainerPort{
		Name:          port.Name,
		ContainerPort: port.Port,
		Protocol:      port.Protocol,
	}
}
