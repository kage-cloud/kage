package kconfig

import (
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
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
	// If the context is a blank string, the current config is returned.
	Api(context string) (kubernetes.Interface, error)
	RestConfig() *rest.Config
	InCluster() bool
	GetNamespace() string
	Raw() *api.Config
}

type ConfigSpec struct {
	ConfigPath string
	Namespace  string
}

type config struct {
	Config      *api.Config
	Interface   kubernetes.Interface
	IsInCluster bool
	Namespace   string
	Rest        *rest.Config
}

func (c *config) RestConfig() *rest.Config {
	return c.Rest
}

func (c *config) GetNamespace() string {
	if c.Namespace == "" {
		f, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return ""
		}
		c.Namespace = string(f)
	}
	return c.Namespace
}

func (c *config) Raw() *api.Config {
	return c.Config
}

func (c *config) InCluster() bool {
	return c.IsInCluster
}

func (c *config) Api(context string) (kubernetes.Interface, error) {
	if c.Interface == nil {
		override := &clientcmd.ConfigOverrides{}
		if context != "" {
			override.CurrentContext = context
		}

		conf, err := clientcmd.NewDefaultClientConfig(
			*c.Config,
			override,
		).ClientConfig()
		if err != nil {
			return nil, err
		}

		return kubernetes.NewForConfig(conf)
	}

	return c.Interface, nil
}

func FromApiConfig(apiConf *api.Config) (Config, error) {
	conf, err := clientcmd.NewDefaultClientConfig(
		*apiConf,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()

	if err != nil {
		return nil, err
	}

	inter, err := kubernetes.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	return &config{
		Config:    apiConf,
		Interface: inter,
		Rest:      conf,
	}, nil
}

func NewConfigClient(spec ConfigSpec) (Config, error) {
	confClient := new(config)

	conf, err := rest.InClusterConfig()
	if err != nil {
		if err == rest.ErrNotInCluster {
			conf, kubeConf, err := loadKubeConfig(spec.ConfigPath)
			if err != nil {
				return nil, err
			}
			confClient.Config = conf
			confClient.Rest, err = kubeConf.ClientConfig()
			if err != nil {
				return nil, err
			}
			if spec.Namespace == "" {
				spec.Namespace, _, err = kubeConf.Namespace()
				if err != nil {
					return nil, err
				}
			}
			confClient.Namespace = spec.Namespace
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

func loadKubeConfig(configPath string) (*api.Config, clientcmd.ClientConfig, error) {
	hd, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	if configPath == "" {
		configPath = path.Join(hd, ".kube", "config")
	}
	clientConf := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: configPath},
		&clientcmd.ConfigOverrides{},
	)

	conf, err := clientConf.RawConfig()
	if err != nil {
		return nil, nil, err
	}

	return &conf, clientConf, err
}
