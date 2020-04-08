package pkg

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/controller"
	"github.com/kage-cloud/kage/xds/pkg/controlplane"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/service"
)

func InjectorFactory() axon.Injector {
	return axon.NewInjector(axon.NewBinder(
		new(controlplane.Package),
		new(service.Package),
		new(factory.Package),
		new(config.Package),
		new(controller.Package),
		new(Package),
	))
}
