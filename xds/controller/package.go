package controller

import "github.com/eddieowens/axon"

const ControllersKey = "Controllers"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(TrafficControllerKey).To().StructPtr(new(trafficController)),
		axon.Bind(ControllersKey).To().Keys(TrafficControllerKey),
	}
}
