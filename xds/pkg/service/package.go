package service

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
)

type Package struct {
}

const KubeClientKey = "KubeClient"

const PersistentEnvoyStateStoreKey = "PersistentEnvoyStateStore"

const StoreClientKey = "StoreClient"

func kubeClientFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	conf := inj.GetStructPtr(config.ConfigKey).(*config.Config)
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

func persistentEnvoyStoreFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	conf := inj.GetStructPtr(config.ConfigKey).(*config.Config)
	var persStore store.EnvoyStatePersistentStore
	if conf.Kube.Namespace == "" {
		client := inj.GetStructPtr(KubeClientKey).(kube.Client)
		spec := &store.KubeStoreSpec{
			Interface: client.Api(),
			Namespace: conf.Kube.Namespace,
		}
		var err error
		persStore, err = store.NewKubeStore(spec)
		if err != nil {
			panic(err)
		}
	} else {
		persStore = store.NewInMemoryStore()
	}

	return axon.StructPtr(persStore)
}

func storeClientFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	persStore := inj.GetStructPtr(PersistentEnvoyStateStoreKey).(store.EnvoyStatePersistentStore)
	spec := &snap.StoreClientSpec{
		PersistentStore: persStore,
	}

	s, err := snap.NewStoreClient(spec)
	if err != nil {
		panic(err)
	}

	return axon.StructPtr(s)
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(EndpointsControllerServiceKey).To().StructPtr(new(endpointsControllerService)),
		axon.Bind(KageServiceKey).To().StructPtr(new(kageService)),
		axon.Bind(CanaryServiceKey).To().StructPtr(new(canaryService)),
		axon.Bind(EnvoyStateServiceKey).To().StructPtr(new(envoyStateService)),
		axon.Bind(XdsEventHandlerKey).To().StructPtr(new(xdsEventHandler)),
		axon.Bind(XdsServiceKey).To().StructPtr(new(xdsService)),
		axon.Bind(WatchServiceKey).To().StructPtr(new(watchService)),
		axon.Bind(CanaryControllerServiceKey).To().StructPtr(new(canaryControllerService)),
		axon.Bind(StateSyncServiceKey).To().StructPtr(new(stateSyncService)),
		axon.Bind(MeshConfigServiceKey).To().StructPtr(new(meshConfigService)),
		axon.Bind(KubeClientKey).To().Factory(kubeClientFactory).WithoutArgs(),
		axon.Bind(LockdownServiceKey).To().Factory(lockDownServiceFactory).WithoutArgs(),
		axon.Bind(KageMeshServiceKey).To().Factory(kageMeshFactory).WithoutArgs(),
		axon.Bind(PersistentEnvoyStateStoreKey).To().Factory(persistentEnvoyStoreFactory).WithoutArgs(),
		axon.Bind(StoreClientKey).To().Factory(storeClientFactory).WithoutArgs(),
	}
}
