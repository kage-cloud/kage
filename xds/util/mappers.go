package util

import (
	"github.com/eddieowens/kage/xds/except"
	envcore "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
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
