package service

import (
	"encoding/json"
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sync"
)

const LockdownServiceKey = "LockdownService"

type LockdownService interface {
	// Removes the selector from the service stopping it from editing the endpoints file.
	LockdownService(svc *corev1.Service, opt kconfig.Opt) error

	// Re-adds the removed selector to the service allowing it to go back to editing the endpoints file.
	ReleaseService(svc *corev1.Service, opt kconfig.Opt) error

	GetLockedDownServices(opt kconfig.Opt) ([]corev1.Service, error)

	IsLockedDown(obj metav1.Object) bool
}

type lockdownService struct {
	KubeClient        kube.Client  `inject:"KubeClient"`
	WatchService      WatchService `inject:"WatchService"`
	selectorsByDeploy map[string]labels.Set
	lock              sync.RWMutex
}

func (l *lockdownService) GetLockedDownServices(opt kconfig.Opt) ([]corev1.Service, error) {
	return l.KubeClient.ListServices(l.lockDownSelector(), opt)
}

func (l *lockdownService) LockdownService(svc *corev1.Service, opt kconfig.Opt) error {
	if l.IsLockedDown(svc) {
		return nil
	}
	if svc.Spec.Selector == nil {
		return except.NewError("service %s's endpoints are already being managed by something else", except.ErrConflict, svc.Name)
	}

	lockdown := &model.Lockdown{DeletedSet: svc.Spec.Selector}

	svc.Spec.Selector = nil

	l.saveLockdownMeta(svc, lockdown)

	if _, err := l.KubeClient.UpdateService(svc, opt); err != nil {
		return err
	}

	return nil
}

func (l *lockdownService) ReleaseService(svc *corev1.Service, opt kconfig.Opt) error {
	deepCopy := svc.DeepCopy()
	lockdown := l.getLockDownMeta(deepCopy)

	deepCopy.Spec.Selector = lockdown.DeletedSet
	l.removeLockdownMeta(deepCopy)
	if _, err := l.KubeClient.UpdateService(deepCopy, opt); err != nil {
		return err
	}
	return nil
}

func (l *lockdownService) IsLockedDown(obj metav1.Object) bool {
	return l.lockDownSelector().Matches(labels.Set(obj.GetLabels()))
}

func (l *lockdownService) lockDownSelector() labels.Selector {
	return labels.SelectorFromSet(map[string]string{
		consts.LabelKeyLockedDown: "true",
	})
}

func (l *lockdownService) saveLockdownMeta(obj metav1.Object, lockdown *model.Lockdown) {
	lbls := obj.GetLabels()
	if lbls == nil {
		lbls = map[string]string{}
	}

	annos := obj.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}

	b, err := json.Marshal(lockdown)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to marshal lockdown %v for %s in %s", lockdown, obj.GetName(), obj.GetNamespace()))
		return
	}

	annos[consts.AnnotationKeyLockdown] = string(b)
	lbls[consts.LabelKeyLockedDown] = "true"

	obj.SetAnnotations(annos)
	obj.SetLabels(lbls)
}

func (l *lockdownService) updateLockdownMeta(deployment *appsv1.Deployment, opt kconfig.Opt) (*model.Lockdown, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	set, ok := l.selectorsByDeploy[deployment.Name]
	if !ok {
		fmt.Println("could not find any labels keys for deploy", deployment.Name)
		return &model.Lockdown{}, nil
	}

	labelKeys := make([]string, len(set))
	i := 0
	for k := range set {
		labelKeys[i] = k
		i++
	}
	l.lock.RUnlock()
	lockdown := &model.Lockdown{
		DeletedSet: map[string]string{},
	}
	depCopy := deployment.DeepCopy()
	for _, key := range labelKeys {
		if v, ok := depCopy.Spec.Template.Labels[key]; ok {
			lockdown.DeletedSet[key] = v
		}
	}

	l.saveLockdownMeta(depCopy, lockdown)

	if _, err := l.KubeClient.UpdateDeploy(depCopy, opt); err != nil {
		return nil, err
	}

	return lockdown, nil
}

func (l *lockdownService) getLockDownMeta(obj metav1.Object) *model.Lockdown {
	if v, ok := obj.GetAnnotations()[consts.AnnotationKeyLockdown]; ok {
		lockdown := new(model.Lockdown)
		if err := json.Unmarshal([]byte(v), lockdown); err != nil {
			return nil
		}
		return lockdown
	}
	return nil
}

func (l *lockdownService) removeLockdownMeta(obj metav1.Object) {
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

func lockDownServiceFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&lockdownService{
		selectorsByDeploy: map[string]labels.Set{},
		lock:              sync.RWMutex{},
	})
}