package main

import (
	"github.com/kage-cloud/kage/xds/pkg"
	"github.com/kage-cloud/kage/xds/pkg/config"
	log "github.com/sirupsen/logrus"
)

func main() {
	injector := pkg.InjectorFactory()
	conf := injector.GetStructPtr(config.ConfigKey).(*config.Config)

	format := &log.TextFormatter{
		TimestampFormat: conf.Log.TimeFormat,
	}

	log.SetFormatter(format)

	logLvl, err := log.ParseLevel(conf.Log.Level)
	if err != nil {
		logLvl = log.InfoLevel
	}

	log.SetLevel(logLvl)
	log.WithField("level", logLvl).
		WithField("time_format", conf.Log.TimeFormat).
		Info("Logger configured.")

	log.Fatal(injector.GetStructPtr(pkg.AppKey).(pkg.App).Start())
}
