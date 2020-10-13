package service

import (
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const ProxyServiceKey = "ProxyService"

var proxySelector = labels.SelectorFromValidatedSet(meta.ToMap(&meta.ProxyMarker{Proxied: true}))

type ProxyService interface {
	// Removes the selector from the service stopping it from editing the endpoints file.
	ProxyService(svc *corev1.Service, replacement labels.Set) error

	// Re-adds the removed selector to the service allowing it to go back to editing the endpoints file.
	ReleaseService(svc *corev1.Service, opt kconfig.Opt) error

	GetSelector(svc *corev1.Service) (labels.Selector, error)

	GetProxiedServices(opt kconfig.Opt) ([]corev1.Service, error)

	GetProxy(svc *corev1.Service) (*meta.Proxy, error)

	IsProxied(obj metav1.Object) bool
}

type proxyService struct {
	KubeClient        kube.Client       `inject:"KubeClient"`
	KubeReaderService KubeReaderService `inject:"KubeReaderService"`
	WatchService      WatchService      `inject:"WatchService"`
}

func (l *proxyService) GetSelector(svc *corev1.Service) (labels.Selector, error) {
	sel := svc.Spec.Selector
	if sel == nil {
		ld, err := l.GetProxy(svc)
		if err == nil {
			sel = ld.DeletedSelector
		}
	}
	return labels.SelectorFromValidatedSet(sel), nil
}

func (l *proxyService) GetProxy(svc *corev1.Service) (*meta.Proxy, error) {
	ld := l.getLockDownMeta(svc)
	if ld == nil {
		return nil, except.NewError("%s does not have a lockdown annotation", except.ErrNotFound, svc.Name)
	}
	return ld, nil
}

func (l *proxyService) GetProxiedServices(opt kconfig.Opt) ([]corev1.Service, error) {
	return l.KubeReaderService.ListServices(proxySelector, opt)
}

func (l *proxyService) ProxyService(svc *corev1.Service, replacement labels.Set) error {
	opt := kconfig.Opt{Namespace: svc.Namespace}

	if l.IsProxied(svc) {
		return nil
	}

	lockdown := &meta.Proxy{DeletedSelector: svc.Spec.Selector, ProxyMarker: meta.ProxyMarker{Proxied: true}}

	svc.Spec.Selector = replacement

	l.saveProxyMeta(svc, lockdown)

	if _, err := l.KubeClient.UpdateService(svc, opt); err != nil {
		return err
	}

	log.WithField("name", svc.Name).WithField("namespace", svc.Namespace).Debug("Locked down service.")

	return nil
}

func (l *proxyService) ReleaseService(svc *corev1.Service, opt kconfig.Opt) error {
	deepCopy := svc.DeepCopy()
	lockdown := l.getLockDownMeta(deepCopy)

	deepCopy.Spec.Selector = lockdown.DeletedSelector
	l.removeLockdownMeta(deepCopy)
	if _, err := l.KubeClient.UpdateService(deepCopy, opt); err != nil {
		return err
	}
	return nil
}

func (l *proxyService) IsProxied(obj metav1.Object) bool {
	return proxySelector.Matches(labels.Set(obj.GetLabels()))
}

func (l *proxyService) saveProxyMeta(obj metav1.Object, lockdown *meta.Proxy) {
	obj.SetAnnotations(meta.Merge(obj.GetAnnotations(), lockdown))
	obj.SetLabels(meta.Merge(obj.GetLabels(), &lockdown.ProxyMarker))
}

func (l *proxyService) getLockDownMeta(obj metav1.Object) *meta.Proxy {
	lockdown := new(meta.Proxy)
	if err := meta.FromMap(obj.GetAnnotations(), lockdown); err != nil {
		return nil
	}
	if err := meta.FromMap(obj.GetLabels(), lockdown); err != nil {
		return nil
	}
	if !lockdown.Proxied {
		return nil
	}
	return lockdown
}

func (l *proxyService) removeLockdownMeta(obj metav1.Object) {
	lbls := obj.GetLabels()

	annos := obj.GetAnnotations()

	if _, ok := lbls[consts.LabelKeyLockedDown]; ok {
		delete(lbls, consts.LabelKeyLockedDown)
	}

	if _, ok := annos[consts.AnnotationKeyLockdown]; ok {
		delete(annos, consts.AnnotationKeyLockdown)
	}

	obj.SetLabels(lbls)
	obj.SetAnnotations(annos)
}
