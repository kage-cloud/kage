package main

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/xds/factory"
	"github.com/kage-cloud/kage/xds/snap"
	"github.com/kage-cloud/kage/xds/handler"
)

func InjectorFactory() axon.Injector {
	return axon.NewInjector(axon.NewBinder(
		new(snap.Package),
		new(factory.Package),
		new(handler.Package),
		new(Package),
	))
}
