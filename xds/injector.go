package main

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/xds/config"
	"github.com/kage-cloud/kage/xds/controlplane"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/service"
	"github.com/kage-cloud/kage/xds/snap"
)

func InjectorFactory() axon.Injector {
	return axon.NewInjector(axon.NewBinder(
		new(snap.Package),
		new(controlplane.Package),
		new(service.Package),
		new(factory.Package),
		new(config.Package),
		new(Package),
	))
}
