package main

import (
	"github.com/kage-cloud/kage/xds/pkg"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Fatal(pkg.InjectorFactory().GetStructPtr(pkg.AppKey).(pkg.App).Start())
}
