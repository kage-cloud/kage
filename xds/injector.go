package main

import (
	"github.com/eddieowens/axon"
	"github.com/eddieowens/kage/xds/factory"
	"github.com/eddieowens/kage/xds/snap"
	"github.com/eddieowens/kage/xds/watcher"
)

func InjectorFactory() axon.Injector {
	return axon.NewInjector(axon.NewBinder(
		new(snap.Package),
		new(factory.Package),
		new(watcher.Package),
		new(Package),
	))
}
