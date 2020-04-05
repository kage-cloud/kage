package main

import log "github.com/sirupsen/logrus"

func main() {
	log.Fatal(InjectorFactory().GetStructPtr(AppKey).(App).Start())
}
