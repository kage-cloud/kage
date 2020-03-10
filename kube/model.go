package kube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Stopper interface {
	Stop(err error)
}

type OnStopFunc func(err error)

func NewStopper(onStop OnStopFunc) Stopper {
	return &stopper{
		onStop: onStop,
	}
}

type stopper struct {
	onStop OnStopFunc
}

func (s stopper) Stop(err error) {
	if s.onStop != nil {
		s.onStop(err)
	}
}

type Filter func(object metav1.Object) bool
