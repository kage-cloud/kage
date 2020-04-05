package service

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/xds/config"
)

type Package struct {
}

const KubeClientKey = "KubeClient"

func kubeClientFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	conf := inj.GetStructPtr("Config").(*config.Config)
	spec := kube.ClientSpec{
		Config: kconfig.ConfigSpec{
			ConfigPath: conf.Kube.Config,
		},
		Context: conf.Kube.Context,
	}

	k, err := kube.NewClient(spec)
	if err != nil {
		panic(err)
	}
	return axon.StructPtr(k)
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(EndpointsControllerServiceKey).To().StructPtr(new(endpointsControllerService)),
		axon.Bind(KageServiceKey).To().StructPtr(new(kageService)),
		axon.Bind(CanaryServiceKey).To().StructPtr(new(canaryService)),
		axon.Bind(EnvoyStateServiceKey).To().StructPtr(new(envoyStateService)),
		axon.Bind(XdsEventHandlerServiceKey).To().StructPtr(new(xdsEventHandler)),
		axon.Bind(XdsServiceKey).To().StructPtr(new(xdsService)),
		axon.Bind(WatchServiceKey).To().StructPtr(new(watchService)),
		axon.Bind(CanaryControllerServiceKey).To().StructPtr(new(canaryControllerService)),
		axon.Bind(KubeClientKey).To().Factory(kubeClientFactory).WithoutArgs(),
		axon.Bind(LockdownServiceKey).To().Factory(lockDownServiceFactory).WithoutArgs(),
		axon.Bind(KageMeshServiceKey).To().Factory(kageMeshFactory).WithoutArgs(),
	}
}
