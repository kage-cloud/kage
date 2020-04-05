package service

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/kube"
)

type Package struct {
}

const KubeClientKey = "KubeClient"

func kubeClientFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	k, err := kube.NewClient()
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
		axon.Bind(KubeClientKey).To().Factory(kubeClientFactory).WithoutArgs(),
		axon.Bind(LockdownServiceKey).To().Factory(lockDownServiceFactory).WithoutArgs(),
		axon.Bind(KageMeshServiceKey).To().Factory(kageMeshFactory).WithoutArgs(),
	}
}
