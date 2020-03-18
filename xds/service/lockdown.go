package service

import (
	"encoding/json"
	"fmt"
	"github.com/eddieowens/axon"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/model"
	"github.com/eddieowens/kage/xds/model/consts"
	"github.com/eddieowens/kage/xds/util/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"sync"
)

const LockdownServiceKey = "LockdownService"

// TODO: Add call to DeployWatchService here.
type LockdownService interface {
	Lockdown(deployment *appsv1.Deployment) error
	DeployPodsEventHandler(deployment *appsv1.Deployment) model.InformEventHandler
	DeployServicesEventHandler(deployment *appsv1.Deployment) model.InformEventHandler
}

type lockdownService struct {
	KubeClient        kube.Client        `inject:"KubeClient"`
	WatchService      DeployWatchService `inject:"DeployWatchService"`
	selectorsByDeploy map[string]labels.Set
	lock              sync.RWMutex
}

func (l *lockdownService) Lockdown(deployment *appsv1.Deployment) error {
	handler := l.DeployServicesEventHandler(deployment)
	if err := l.WatchService.DeploymentServices(deployment, handler); err != nil {
		return err
	}

	if err := l.WatchService.DeploymentPods(deployment, l.DeployPodsEventHandler(deployment)); err != nil {
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
				for k := range lockdown.DeletedLabels {
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
	lockdown := model.Lockdown{
		DeletedLabels: map[string]string{},
	}
	depCopy := deployment.DeepCopy()
	for _, key := range labelKeys {
		if v, ok := depCopy.Spec.Template.Labels[key]; ok {
			lockdown.DeletedLabels[key] = v
		}
	}

	if depCopy.Annotations == nil {
		depCopy.Annotations = map[string]string{}
	}

	if depCopy.Labels == nil {
		depCopy.Labels = map[string]string{}
	}

	b, err := json.Marshal(lockdown)
	if err != nil {
		return nil, err
	}

	depCopy.Annotations[consts.AnnotationKeyLockdown] = string(b)
	depCopy.Labels[consts.LabelKeyLockedDown] = "true"

	if _, err := l.KubeClient.UpdateDeploy(depCopy, opt); err != nil {
		return nil, err
	}

	return &lockdown, nil
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
					l.selectorsByDeploy[deployment.Name] = l.mergeSets(v, svc.Spec.Selector)
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
					l.selectorsByDeploy[deployment.Name] = l.mergeSets(v, svc.Spec.Selector)
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

// Merges Set s2 into Set s1 and returns the resulting set.
func (l *lockdownService) mergeSets(s1 labels.Set, s2 labels.Set) labels.Set {
	for k, v := range s2 {
		if _, ok := s1[k]; !ok {
			s1[k] = v
		}
	}
	return s1
}

func lockDownServiceFactory(_ axon.Injector, _ axon.Args) axon.Instance {
	return axon.StructPtr(&lockdownService{
		selectorsByDeploy: map[string]labels.Set{},
		lock:              sync.RWMutex{},
	})
}
