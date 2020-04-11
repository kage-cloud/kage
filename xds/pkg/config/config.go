package config

import (
	"bytes"
	"github.com/eddieowens/axon"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
)

const ConfigKey = "Config"

type Config struct {
	Server Server `mapstructure:"server"`
	Kube   Kube   `mapstructure:"kube"`
	Xds    Xds    `mapstructure:"xds"`
	Log    Log    `mapstructure:"log"`
}

type Log struct {
	Level      string `mapstructure:"level"`
	TimeFormat string `mapstructure:"timeformat"`
}

type Server struct {
	Port uint16 `mapstructure:"port"`
}

type Kube struct {
	Config    string `mapstructure:"config"`
	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
}

type Xds struct {
	Port      uint16 `mapstructure:"port"`
	Address   string `mapstructure:"address"`
	AdminPort uint16 `mapstructure:"adminport"`
}

func defaultConfig() *Config {
	return &Config{
		Server: Server{
			Port: 8080,
		},
		Xds: Xds{
			Port:      8081,
			Address:   "0.0.0.0",
			AdminPort: 8082,
		},
		Kube: Kube{
			Config: clientcmd.RecommendedHomeFile,
		},
	}
}

func configFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("kage")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AllowEmptyEnv(false)

	b, _ := yaml.Marshal(defaultConfig())
	defaultConfig := bytes.NewReader(b)
	if err := v.MergeConfig(defaultConfig); err != nil {
		panic(err)
	}

	configPath := os.Getenv("KUBE_CONFIG_PATH")
	v.SetConfigFile(configPath)
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.WithField("path", configPath).WithError(err).Debug("Failed to load config file")
		} else {
			panic(err)
		}
	}

	v.AutomaticEnv()

	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		log.Fatal(err)
	}

	return axon.Any(config)
}
