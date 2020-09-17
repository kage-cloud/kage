package util

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/kage-cloud/kage/core/except"
	corev1 "k8s.io/api/core/v1"
)

func KubeProtocolToSocketAddressProtocol(protocol corev1.Protocol) (core.SocketAddress_Protocol, error) {
	switch protocol {
	case corev1.ProtocolSCTP:
		return -1, except.NewError("SCTP is not a supported protocol", except.ErrUnsupported)
	case corev1.ProtocolTCP:
		return core.SocketAddress_TCP, nil
	case corev1.ProtocolUDP:
		return core.SocketAddress_UDP, nil
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
