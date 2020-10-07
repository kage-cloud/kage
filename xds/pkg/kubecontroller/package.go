package kubecontroller

import "github.com/eddieowens/axon"

const KubeControllersKey = "KubeControllers"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(EnvoyKubeControllerKey).To().StructPtr(new(Envoy)),
		axon.Bind(EndpointsKubeControllerKey).To().StructPtr(new(Endpoints)),
		axon.Bind(KubeControllersKey).To().Keys(EnvoyKubeControllerKey, EndpointsKubeControllerKey),
	}
}
