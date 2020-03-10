package service

import (
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const EndpointFinderServiceKey = "EndpointFinderService"

type EndpointFinderService interface {
	FindEndpoints(dep *appsv1.Deployment, opt kconfig.Opt) ([]corev1.Endpoints, error)
}

type endpointFinderService struct {
	KubeClient kube.Client `inject:"KubeClient"`
}

func (e *endpointFinderService) FindEndpoints(dep *appsv1.Deployment, opt kconfig.Opt) ([]corev1.Endpoints, error) {
	lo := metav1.ListOptions{}

	svcs, err := e.KubeClient.ListServices(lo, opt)
	if err != nil {
		return nil, err
	}

	matchingSvcs := make([]corev1.Service, 0)
	for _, s := range svcs {
		selector := labels.SelectorFromSet(s.Spec.Selector)
		if selector.Matches(labels.Set(dep.Spec.Template.Labels)) {
			matchingSvcs = append(matchingSvcs, s)
		}
	}

	eps := make([]corev1.Endpoints, 0)
	for _, s := range matchingSvcs {
		ep, err := e.KubeClient.GetEndpoints(s.Name, opt)
		if err != nil {
			return nil, err
		}

		eps = append(eps, ep)
	}

	return eps, nil
}
