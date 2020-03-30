package envoyutil

import (
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	corev2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
)

func ListenerMatchesPort(port uint32, listener *apiv2.Listener) bool {
	if v, ok := listener.Address.Address.(*corev2.Address_SocketAddress); ok {
		if ps, ok := v.SocketAddress.PortSpecifier.(*corev2.SocketAddress_PortValue); ok {
			if ps.PortValue == port {
				return true
			}
		}
	}
	return false
}

func FindListenerPort(port uint32, listeners []apiv2.Listener) (*apiv2.Listener, int) {
	for i, v := range listeners {
		if ListenerMatchesPort(port, &v) {
			return &v, i
		}
	}
	return nil, -1
}

func ContainsListenerPort(port uint32, listeners []apiv2.Listener) bool {
	v, _ := FindListenerPort(port, listeners)
	return v != nil
}

func RemoveListenerPort(port uint32, listeners []apiv2.Listener) []apiv2.Listener {
	v, idx := FindListenerPort(port, listeners)
	if v != nil {
		return append(listeners[:idx], listeners[idx+1:]...)
	}
	return listeners
}
