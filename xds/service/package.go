package service

import "github.com/eddieowens/axon"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(TrafficControllerServiceKey).To().StructPtr(new(trafficControllerService)),
		axon.Bind(EndpointsControllerServiceKey).To().StructPtr(new(endpointsControllerService)),
		axon.Bind(LockdownServiceKey).To().Factory(lockDownServiceFactory).WithoutArgs(),
		axon.Bind(KageMeshServiceKey).To().Factory(kageMeshFactory).WithoutArgs(),
	}
}
