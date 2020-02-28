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
}

type config struct {
	ApiConfig *api.Config
	Interface kubernetes.Interface
}

func (c *config) Api(context string) (kubernetes.Interface, error) {
	if c.Interface == nil {
		conf, err := clientcmd.NewDefaultClientConfig(
			*c.ApiConfig,
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

func NewConfigClient() (Config, error) {
	confClient := new(config)

	conf, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			conf, err := loadKubeConfig()
			confClient.ApiConfig = conf
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		confClient.Interface, err = kubernetes.NewForConfig(conf)
		if err != nil {
			return nil, err
		}
	}

	return confClient, nil
}

func loadKubeConfig() (*api.Config, error) {
	hd, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	conf, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: path.Join(hd, ".kube", "config")},
		&clientcmd.ConfigOverrides{},
	).RawConfig()

	return &conf, err
}
