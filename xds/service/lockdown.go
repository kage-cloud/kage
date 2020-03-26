package service

import (
	"encoding/json"
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/kage-cloud/kage/kube"
	"github.com/kage-cloud/kage/kube/kconfig"
	"github.com/kage-cloud/kage/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/except"
	"github.com/kage-cloud/kage/xds/model"
	"github.com/kage-cloud/kage/xds/model/consts"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sync"
)

const LockdownServiceKey = "LockdownService"

type LockdownService interface {
	// Incomplete
	LockdownDeploy(deployment *appsv1.Deployment) error
	// Incomplete
	ReleaseDeploy(deployment *appsv1.Deployment) error

	// Removes the selector from the service stopping it from editing the endpoints file.
	LockdownService(svc *corev1.Service, opt kconfig.Opt) error

	// Re-adds the removed selector to the service allowing it to go back to editing the endpoints file.
	ReleaseService(svc *corev1.Service, opt kconfig.Opt) error

	IsLockedDown(obj metav1.Object) bool
}

type lockdownService struct {
	KubeClient            kube.Client           `inject:"KubeClient"`
	WatchService          DeployWatchService    `inject:"DeployWatchService"`
	StopperHandlerService StopperHandlerService `inject:"StopperHandlerService"`
	selectorsByDeploy     map[string]labels.Set
	lock                  sync.RWMutex
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
	if v, ok := obj.GetLabels()[consts.LabelKeyLockedDown]; ok {
		return v == "true"
	}
	return false
}

func (l *lockdownService) ReleaseDeploy(deployment *appsv1.Deployment) error {
	lockdown := l.getLockDownMeta(deployment)
	if lockdown == nil {
		return nil
	}

	opt := kconfig.Opt{Namespace: kubeutil.DeploymentPodNamespace(deployment)}

	_, err := l.rollbackLockdown(deployment, lockdown, opt)
	if err != nil {
		fmt.Println("Failed to rollback the lockdown of", deployment.Name, "in", opt.Namespace, ":", err.Error())
		return err
	}

	return nil
}

func (l *lockdownService) LockdownDeploy(deployment *appsv1.Deployment) error {
	err := l.WatchService.DeploymentServices(deployment, l.DeployServicesEventHandler(deployment))
	if err != nil {
		return err
	}

	err = l.WatchService.DeploymentPods(deployment, l.DeployPodsEventHandler(deployment))
	if err != nil {
		return err
	}
	return nil
}

func (l *lockdownService) DeployServicesEventHandler(deployment *appsv1.Deployment) model.InformEventHandler {
	return &model.InformEventHandlerFuncs{
		OnWatch: l.watchSvcEvent(deployment),
		OnList:  l.listSvcEvent(deployment),
	}
}

func (l *lockdownService) DeployPodsEventHandler(deployment *appsv1.Deployment) model.InformEventHandler {
	return &model.InformEventHandlerFuncs{
		OnWatch: l.watchPodEvent(deployment),
		OnList:  l.listPodEvent(deployment),
	}
}

func (l *lockdownService) listPodEvent(deployment *appsv1.Deployment) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.PodList); ok {

			opt := kconfig.Opt{Namespace: deployment.Namespace}

			lockdown, err := l.updateLockdownMeta(deployment, opt)
			if err != nil {
				return err
			}

			for _, p := range v.Items {
				for k := range lockdown.DeletedSet {
					delete(p.Labels, k)
				}
				if _, err := l.KubeClient.UpdatePod(&p, opt); err != nil {
					return err
				}
			}
		}
		return nil
	}
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

	// TODO: Remove the labels from the pod template spec as well.

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

func (l *lockdownService) watchPodEvent(deployment *appsv1.Deployment) model.OnWatchEventFunc {
	return func(event watch.Event) {
		switch event.Type {
		case watch.Added, watch.Modified:
			if p, ok := event.Object.(*corev1.Pod); ok {
				l.lock.RLock()
				defer l.lock.RUnlock()
				set, ok := l.selectorsByDeploy[deployment.Name]
				if !ok {
					fmt.Println("could not find any labels keys for deploy", deployment.Name)
					break
				}
				labelKeys := make([]string, len(set))
				i := 0
				for k := range set {
					labelKeys[i] = k
					i++
				}
				opt := kconfig.Opt{Namespace: kubeutil.DeploymentPodNamespace(deployment)}
				for _, key := range labelKeys {
					if _, ok = p.Labels[key]; ok {
						delete(p.Labels, key)
					}
				}
				if _, err := l.KubeClient.UpdatePod(p, opt); err != nil {
					fmt.Println("Failed to lockdown pod", p.Name, "in", p.Namespace, "for deploy", deployment.Name, ":", err.Error())
				}
			}
		}
	}
}

func (l *lockdownService) listSvcEvent(deployment *appsv1.Deployment) model.OnListEventFunc {
	return func(obj runtime.Object) error {
		if v, ok := obj.(*corev1.ServiceList); ok {
			l.lock.Lock()
			defer l.lock.Unlock()
			for _, svc := range v.Items {
				if v, ok := l.selectorsByDeploy[deployment.Name]; ok {
					l.selectorsByDeploy[deployment.Name] = labels.Merge(v, svc.Spec.Selector)
				}
			}
		}
		return nil
	}
}

func (l *lockdownService) watchSvcEvent(deployment *appsv1.Deployment) model.OnWatchEventFunc {
	return func(event watch.Event) {
		switch event.Type {
		case watch.Added, watch.Modified:
			if svc, ok := event.Object.(*corev1.Service); ok {
				l.lock.Lock()
				defer l.lock.Unlock()
				if v, ok := l.selectorsByDeploy[deployment.Name]; ok {
					l.selectorsByDeploy[deployment.Name] = labels.Merge(v, svc.Spec.Selector)
				}
				l.lock.Unlock()
				opt := kconfig.Opt{Namespace: deployment.Namespace}
				if _, err := l.updateLockdownMeta(deployment, opt); err != nil {
					fmt.Println("Failed to update the lockdown for", deployment.Name, "when service", svc.Name, "was updated in", deployment.Namespace)
				}
			}
			break
		}
	}
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

func (l *lockdownService) rollbackLockdown(deploy *appsv1.Deployment, lockdown *model.Lockdown, opt kconfig.Opt) (*appsv1.Deployment, error) {
	dep := deploy.DeepCopy()

	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, err
	}

	depPods, err := l.KubeClient.ListPods(selector.String(), opt)
	if err != nil {
		fmt.Println("Failed to rollback the lockdown of", deploy.Name, "in", opt.Namespace, ":", err.Error())
		return nil, err
	}

	for _, p := range depPods {
		for k := range lockdown.DeletedSet {
			delete(p.Labels, k)
		}
		if _, err := l.KubeClient.UpdatePod(&p, opt); err != nil {
			fmt.Println("Failed to remove the lockdown for pod", p.Name, "in", opt.Namespace, ":", err.Error())
		}
	}

	l.removeLockdownMeta(dep)

	// TODO: re-add the deleted labels to the pod template spec for the dep

	return dep, nil
}

func lockDownServiceFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&lockdownService{
		selectorsByDeploy: map[string]labels.Set{},
		lock:              sync.RWMutex{},
	})
}
