package kstream

import (
	"github.com/kage-cloud/kage/core/kube/ktypes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type Collector interface {
	Objects() []runtime.Object
	MetaObjects() []metav1.Object
	Deployments() *appsv1.DeploymentList
	Services() *corev1.ServiceList
	ListInterface() metav1.ListInterface

	// If the underlying object's type has a LabelSelector, return them. If the object is a Pod, a Selector which matches
	// that Pod is returned.
	LabelSelectors() []labels.Selector
}

type collector struct {
	Streamer *streamer
}

func (c *collector) Services() *corev1.ServiceList {
	deps := make([]corev1.Service, 0, len(c.Streamer.objs))

	for _, obj := range c.Streamer.objs {
		if v, ok := obj.(*corev1.Service); ok {
			deps = append(deps, *v)
		}
	}

	return &corev1.ServiceList{Items: deps}
}

func (c *collector) ListInterface() metav1.ListInterface {
	return c.Streamer.li
}

func (c *collector) Objects() []runtime.Object {
	return c.Streamer.objs
}

func (c *collector) MetaObjects() []metav1.Object {
	metaObjs := make([]metav1.Object, 0, len(c.Streamer.objs))

	for _, obj := range c.Streamer.objs {
		if v, ok := obj.(metav1.Object); ok {
			metaObjs = append(metaObjs, v)
		}
	}

	return metaObjs
}

func (c *collector) Deployments() *appsv1.DeploymentList {
	deps := make([]appsv1.Deployment, 0, len(c.Streamer.objs))

	for _, obj := range c.Streamer.objs {
		if v, ok := obj.(*appsv1.Deployment); ok {
			deps = append(deps, *v)
		}
	}

	return &appsv1.DeploymentList{Items: deps}
}

func (c *collector) LabelSelectors() []labels.Selector {
	selectors := make([]labels.Selector, 0, len(c.Streamer.objs))

	for _, v := range c.Streamer.objs {
		if v, ok := v.(*corev1.Pod); ok && v.Labels != nil {
			selectors = append(selectors, labels.SelectorFromValidatedSet(v.Labels))
			continue
		}
		ls := ktypes.PodSelectorAsSelector(v)
		if ls != nil {
			selectors = append(selectors, ls)
		}
	}
	return selectors
}
