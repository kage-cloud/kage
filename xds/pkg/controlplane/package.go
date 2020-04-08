package controlplane

import "github.com/eddieowens/axon"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(EnvoyControlPlaneKey).To().StructPtr(new(envoyControlPlane)),
	}
}
