package watcher

import "github.com/eddieowens/axon"

type Package struct {
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(XdsWatcherKey).To().Factory(xdsWatcherFactory).WithoutArgs(),
	}
}
