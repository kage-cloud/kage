package envoyutil

import (
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
)

func ListenerMatchesPort(port uint32, listener *listener.Listener) bool {
	if v, ok := listener.Address.Address.(*core.Address_SocketAddress); ok {
		if ps, ok := v.SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue); ok {
			if ps.PortValue == port {
				return true
			}
		}
	}
	return false
}

func FindListenerPort(port uint32, listeners []listener.Listener) (*listener.Listener, int) {
	for i, v := range listeners {
		if ListenerMatchesPort(port, &v) {
			return &v, i
		}
	}
	return nil, -1
}

func ContainsListenerPort(port uint32, listeners []listener.Listener) bool {
	v, _ := FindListenerPort(port, listeners)
	return v != nil
}

func RemoveListenerPort(port uint32, listeners []listener.Listener) []listener.Listener {
	v, idx := FindListenerPort(port, listeners)
	if v != nil {
		return append(listeners[:idx], listeners[idx+1:]...)
	}
	return listeners
}
