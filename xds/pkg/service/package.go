package service

import (
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	log "github.com/sirupsen/logrus"
)

type Package struct {
}

const KubeClientKey = "KubeClient"

const PersistentEnvoyStateStoreKey = "PersistentEnvoyStateStore"

const StoreClientKey = "StoreClient"

const InformerClientKey = "InformerClient"

func kubeClientFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	conf := inj.GetStructPtr(config.ConfigKey).(*config.Config)
	spec := kube.ClientSpec{
		Config: kconfig.ConfigSpec{
			ConfigPath: conf.Kube.Config,
			Namespace:  conf.Kube.Namespace,
		},
		Context: conf.Kube.Context,
	}

	k, err := kube.NewClient(spec)
	if err != nil {
		panic(err)
	}

	if k.ApiConfig().InCluster() {
		log.Info("Running in cluster mode")
	} else {
		log.WithField("config_path", conf.Kube.Config).Info("Not running in cluster mode")
	}

	log.WithField("context", k.ApiConfig().Raw().CurrentContext).
		WithField("client_version", "1.15.10").
		WithField("namespace", k.ApiConfig().GetNamespace()).
		Info("Configured Kubernetes client")

	return axon.StructPtr(k)
}

func persistentEnvoyStoreFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	client := inj.GetStructPtr(KubeClientKey).(kube.Client)
	var persStore store.EnvoyStatePersistentStore
	if client.ApiConfig().InCluster() {
		spec := &store.KubeStoreSpec{
			Interface: client.Api(),
			Namespace: client.ApiConfig().GetNamespace(),
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

func informerClientFactory(inj axon.Injector, _ axon.Args) axon.Instance {
	client := inj.GetStructPtr(KubeClientKey).(kube.Client)
	return axon.StructPtr(kube.NewInformerClient(client))
}

func (p *Package) Bindings() []axon.Binding {
	return []axon.Binding{
		axon.Bind(EndpointsControllerServiceKey).To().StructPtr(new(endpointsControllerService)),
		axon.Bind(CanaryServiceKey).To().StructPtr(new(canaryService)),
		axon.Bind(EnvoyStateServiceKey).To().StructPtr(new(envoyStateService)),
		axon.Bind(EnvoyEndpointControllerKey).To().StructPtr(new(envoyEndpointController)),
		axon.Bind(XdsServiceKey).To().StructPtr(new(xdsService)),
		axon.Bind(WatchServiceKey).To().StructPtr(new(watchService)),
		axon.Bind(CanaryControllerServiceKey).To().StructPtr(new(canaryControllerService)),
		axon.Bind(StateSyncServiceKey).To().StructPtr(new(stateSyncService)),
		axon.Bind(MeshConfigServiceKey).To().StructPtr(new(meshConfigService)),
		axon.Bind(KageMeshServiceKey).To().StructPtr(new(kageMeshService)),
		axon.Bind(KubeReaderServiceKey).To().StructPtr(new(kubeReaderService)),
		axon.Bind(CanaryEndpointsServiceKey).To().StructPtr(new(canaryEndpointsService)),
		axon.Bind(ProxyServiceKey).To().StructPtr(new(proxyService)),
		axon.Bind(KageServiceKey).To().Factory(kageServiceFactory).WithoutArgs(),
		axon.Bind(KubeClientKey).To().Factory(kubeClientFactory).WithoutArgs(),
		axon.Bind(PersistentEnvoyStateStoreKey).To().Factory(persistentEnvoyStoreFactory).WithoutArgs(),
		axon.Bind(StoreClientKey).To().Factory(storeClientFactory).WithoutArgs(),
		axon.Bind(InformerClientKey).To().Factory(informerClientFactory).WithoutArgs(),
	}
}
