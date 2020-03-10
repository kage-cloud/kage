package factory

import "github.com/eddieowens/axon"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(ListenerFactoryKey).To().StructPtr(NewListenerFactory()),
		axon.Bind(EndpointFactoryKey).To().StructPtr(NewEndpointFactory()),
		axon.Bind(RouteFactoryKey).To().StructPtr(NewRouteFactory()),
		axon.Bind(KageMeshFactoryKey).To().StructPtr(new(kageMeshFactory)),
	}
}
