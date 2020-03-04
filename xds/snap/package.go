package snap

import (
	"github.com/eddieowens/axon"
)

type Package struct {
}

func storeClientFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	sc, err := NewStoreClient()
	if err != nil {
		panic(err)
	}
	return axon.StructPtr(sc)
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(StoreClientKey).To().Factory(storeClientFactory).WithoutArgs(),
	}
}
