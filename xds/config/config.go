package config

import (
	"bytes"
	"github.com/eddieowens/axon"
	"github.com/labstack/gommon/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
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
	Config  string `mapstructure:"config"`
	Context string `mapstructure:"context"`
}

type Xds struct {
	Port uint16 `mapstructure:"port"`
}

func defaultConfig() *Config {
	return &Config{
		Server: Server{
			Port: 8080,
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

	configPath := os.Getenv("KAGE_CONFIG_PATH")
	v.SetConfigFile(configPath)
	if err := v.MergeInConfig(); err != nil {
		panic(err)
	}

	v.AutomaticEnv()

	config := Config{}
	if err := v.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}

	return axon.Any(config)
}
