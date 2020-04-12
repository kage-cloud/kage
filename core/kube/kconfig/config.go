package kconfig

import (
	"io/ioutil"
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
			conf, kubeConf, err := loadKubeConfig(spec.ConfigPath)
			if err != nil {
				return nil, err
			}
			confClient.Config = conf
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
