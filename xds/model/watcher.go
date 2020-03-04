package model

import "k8s.io/apimachinery/pkg/watch"

type OnKubeWatcherEvent interface {
	HandleEvent(p watch.Event)
}
