package kconfig

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path"
)

type Opt struct {
	Context   string
	Namespace string
}

type Config interface {
	Api(context string) (kubernetes.Interface, error)
	InCluster() bool
	ApiConfig() *api.Config
}

type ConfigSpec struct {
	ConfigPath string
}

type config struct {
	Config      *api.Config
	Interface   kubernetes.Interface
	IsInCluster bool
}

func (c *config) ApiConfig() *api.Config {
	return c.Config
}

func (c *config) InCluster() bool {
	return c.IsInCluster
}

func (c *config) Api(context string) (kubernetes.Interface, error) {
	if c.Interface == nil {
		conf, err := clientcmd.NewDefaultClientConfig(
			*c.Config,
			&clientcmd.ConfigOverrides{
				CurrentContext: context,
			},
		).ClientConfig()
		if err != nil {
			return nil, err
		}

		return kubernetes.NewForConfig(conf)
	}

	return c.Interface, nil
}

func NewConfigClient(spec ConfigSpec) (Config, error) {
	confClient := new(config)

	conf, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			conf, err := loadKubeConfig(spec.ConfigPath)
			confClient.Config = conf
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		confClient.Interface, err = kubernetes.NewForConfig(conf)
		confClient.IsInCluster = true
		if err != nil {
			return nil, err
		}
	}

	return confClient, nil
}

func loadKubeConfig(configPath string) (*api.Config, error) {
	hd, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	if configPath == "" {
		configPath = path.Join(hd, ".kube", "config")
	}
	conf, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath},
		&clientcmd.ConfigOverrides{},
	).RawConfig()

	return &conf, err
}
