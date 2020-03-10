package model

import "k8s.io/apimachinery/pkg/watch"

type EventHandler func(watch.Event)
