package main

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/kube"
)

type Package struct {
}

const KubeClientKey = "KubeClient"

func KubeClientFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	kc, err := kube.NewClient()
	if err != nil {
		panic(err)
	}

	return axon.StructPtr(kc)
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(AppKey).To().StructPtr(new(app)),
		axon.Bind(KubeClientKey).To().Factory(KubeClientFactory).WithoutArgs(),
	}
}
