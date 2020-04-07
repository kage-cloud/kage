package controller

import "github.com/eddieowens/axon"

const ControllersKey = "Controllers"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(CanaryControllerKey).To().StructPtr(new(canaryController)),
		axon.Bind(ControllersKey).To().Keys(CanaryControllerKey),
	}
}
