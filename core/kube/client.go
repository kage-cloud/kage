package kube

import (
	"fmt"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
	"time"
)

type Client interface {
	WatchDeploy(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error)
	WaitTillDeployReady(name string, timeout time.Duration, opt kconfig.Opt) error
	DeleteConfigMap(name string, opt kconfig.Opt) error
	UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error)
	UpsertDeploy(dep *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error)
	SaveEndpoints(ep *corev1.Endpoints, opt kconfig.Opt) (*corev1.Endpoints, error)
	UpdatePod(pod *corev1.Pod, opt kconfig.Opt) (*corev1.Pod, error)
	UpdateEndpoints(ep *corev1.Endpoints, opt kconfig.Opt) (*corev1.Endpoints, error)
	UpdateDeploy(deploy *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error)
	DeleteDeploy(name string, opt kconfig.Opt) error
	CreateDeploy(deploy *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error)
	UpdateService(service *corev1.Service, opt kconfig.Opt) (*corev1.Service, error)

	Get(name string, kind ktypes.Kind, opt kconfig.Opt) (runtime.Object, error)
	List(kind ktypes.Kind, options metav1.ListOptions, opt kconfig.Opt) (metav1.ListInterface, error)
	Create(obj runtime.Object, opt kconfig.Opt) (runtime.Object, error)
	Delete(name string, kind ktypes.Kind, opt kconfig.Opt) error
	Update(obj runtime.Object, opt kconfig.Opt) (runtime.Object, error)

	Api() kubernetes.Interface
	ApiConfig() kconfig.Config
}

func FromApiConfig(conf *api.Config) (Client, error) {
	configClient, err := kconfig.FromApiConfig(conf)
	if err != nil {
		return nil, err
	}

	inter, err := configClient.Api("")
	if err != nil {
		return nil, err
	}

	return &client{
		Interface: inter,
		Config:    configClient,
	}, nil
}

func NewClient(spec ClientSpec) (Client, error) {
	conf, err := kconfig.NewConfigClient(spec.Config)
	if err != nil {
		return nil, err
	}

	apiClient, err := conf.Api(spec.Context)
	if err != nil {
		return nil, err
	}

	return &client{
		Interface: apiClient,
		Config:    conf,
	}, nil
}

type ClientSpec struct {
	Config  kconfig.ConfigSpec
	Context string
}

type client struct {
	Interface kubernetes.Interface
	Config    kconfig.Config
}

func (c *client) Create(obj runtime.Object, opt kconfig.Opt) (runtime.Object, error) {
	switch typ := obj.(type) {
	case *corev1.Pod:
		return c.Api().CoreV1().Pods(opt.Namespace).Update(typ)
	case *appsv1.Deployment:
		return c.Api().AppsV1().Deployments(opt.Namespace).Update(typ)
	case *corev1.Service:
		return c.Api().CoreV1().Services(opt.Namespace).Update(typ)
	case *appsv1.ReplicaSet:
		return c.Api().AppsV1().ReplicaSets(opt.Namespace).Update(typ)
	case *corev1.ConfigMap:
		return c.Api().CoreV1().ConfigMaps(opt.Namespace).Update(typ)
	case *corev1.Endpoints:
		return c.Api().CoreV1().Endpoints(opt.Namespace).Update(typ)
	case *appsv1.DaemonSet:
		return c.Api().AppsV1().DaemonSets(opt.Namespace).Update(typ)
	case *appsv1.StatefulSet:
		return c.Api().AppsV1().StatefulSets(opt.Namespace).Update(typ)
	}

	return nil, except.NewError("%T is not a supported Kubernetes kind", except.ErrUnsupported, obj)
}

func (c *client) Delete(name string, kind ktypes.Kind, opt kconfig.Opt) error {
	switch kind {
	case ktypes.KindPod:
		return c.Api().CoreV1().Pods(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindDeployment:
		return c.Api().AppsV1().Deployments(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindService:
		return c.Api().CoreV1().Services(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindReplicaSet:
		return c.Api().AppsV1().ReplicaSets(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindConfigMap:
		return c.Api().CoreV1().ConfigMaps(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindEndpoints:
		return c.Api().CoreV1().Endpoints(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindDaemonSet:
		return c.Api().AppsV1().DaemonSets(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	case ktypes.KindStatefulSet:
		return c.Api().AppsV1().StatefulSets(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
	}

	return except.NewError("%s is not a supported Kubernetes kind", except.ErrUnsupported, kind)
}

func (c *client) Update(obj runtime.Object, opt kconfig.Opt) (runtime.Object, error) {
	switch typ := obj.(type) {
	case *corev1.Pod:
		return c.Api().CoreV1().Pods(opt.Namespace).Create(typ)
	case *appsv1.Deployment:
		return c.Api().AppsV1().Deployments(opt.Namespace).Create(typ)
	case *corev1.Service:
		return c.Api().CoreV1().Services(opt.Namespace).Create(typ)
	case *appsv1.ReplicaSet:
		return c.Api().AppsV1().ReplicaSets(opt.Namespace).Create(typ)
	case *corev1.ConfigMap:
		return c.Api().CoreV1().ConfigMaps(opt.Namespace).Create(typ)
	case *corev1.Endpoints:
		return c.Api().CoreV1().Endpoints(opt.Namespace).Create(typ)
	case *appsv1.DaemonSet:
		return c.Api().AppsV1().DaemonSets(opt.Namespace).Create(typ)
	case *appsv1.StatefulSet:
		return c.Api().AppsV1().StatefulSets(opt.Namespace).Create(typ)
	}

	return nil, except.NewError("%T is not a supported Kubernetes kind", except.ErrUnsupported, obj)
}

func (c *client) List(kind ktypes.Kind, options metav1.ListOptions, opt kconfig.Opt) (metav1.ListInterface, error) {
	switch kind {
	case ktypes.KindPod:
		return c.Api().CoreV1().Pods(opt.Namespace).List(options)
	case ktypes.KindDeployment:
		return c.Api().AppsV1().Deployments(opt.Namespace).List(options)
	case ktypes.KindService:
		return c.Api().CoreV1().Services(opt.Namespace).List(options)
	case ktypes.KindReplicaSet:
		return c.Api().AppsV1().ReplicaSets(opt.Namespace).List(options)
	case ktypes.KindConfigMap:
		return c.Api().CoreV1().ConfigMaps(opt.Namespace).List(options)
	case ktypes.KindEndpoints:
		return c.Api().CoreV1().Endpoints(opt.Namespace).List(options)
	case ktypes.KindDaemonSet:
		return c.Api().AppsV1().DaemonSets(opt.Namespace).List(options)
	case ktypes.KindStatefulSet:
		return c.Api().AppsV1().StatefulSets(opt.Namespace).List(options)
	}

	return nil, except.NewError("%s is not a supported Kubernetes kind", except.ErrUnsupported, kind)
}

func (c *client) Get(name string, kind ktypes.Kind, opt kconfig.Opt) (runtime.Object, error) {
	switch kind {
	case ktypes.KindPod:
		return c.Api().CoreV1().Pods(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindDeployment:
		return c.Api().AppsV1().Deployments(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindService:
		return c.Api().CoreV1().Services(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindReplicaSet:
		return c.Api().AppsV1().ReplicaSets(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindConfigMap:
		return c.Api().CoreV1().ConfigMaps(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindEndpoints:
		return c.Api().CoreV1().Endpoints(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindDaemonSet:
		return c.Api().AppsV1().DaemonSets(opt.Namespace).Get(name, metav1.GetOptions{})
	case ktypes.KindStatefulSet:
		return c.Api().AppsV1().StatefulSets(opt.Namespace).Get(name, metav1.GetOptions{})
	}

	return nil, except.NewError("%s is not a supported Kubernetes kind", except.ErrUnsupported, kind)
}

func (c *client) SaveEndpoints(ep *corev1.Endpoints, opt kconfig.Opt) (*corev1.Endpoints, error) {
	ep.ResourceVersion = ""
	out, err := c.Api().CoreV1().Endpoints(opt.Namespace).Create(ep)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return c.UpdateEndpoints(ep, opt)
		} else {
			return nil, err
		}
	}
	return out, nil
}

func (c *client) ApiConfig() kconfig.Config {
	return c.Config
}

func (c *client) Api() kubernetes.Interface {
	return c.Interface
}

func (c *client) UpdateEndpoints(ep *corev1.Endpoints, opt kconfig.Opt) (*corev1.Endpoints, error) {
	return c.Api().CoreV1().Endpoints(opt.Namespace).Update(ep)
}

func (c *client) CreateDeploy(deploy *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error) {
	deploy.ResourceVersion = ""
	return c.Api().AppsV1().Deployments(opt.Namespace).Create(deploy)
}

func (c *client) UpdateDeploy(deploy *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error) {
	return c.Api().AppsV1().Deployments(opt.Namespace).Update(deploy)
}

func (c *client) UpdatePod(pod *corev1.Pod, opt kconfig.Opt) (*corev1.Pod, error) {
	return c.Api().CoreV1().Pods(opt.Namespace).Update(pod)
}

func (c *client) UpdateService(service *corev1.Service, opt kconfig.Opt) (*corev1.Service, error) {
	return c.Api().CoreV1().Services(opt.Namespace).Update(service)
}

func (c *client) WatchDeploy(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error) {
	return c.Api().AppsV1().Deployments(opt.Namespace).Watch(lo)
}

func (c *client) WaitTillDeployReady(name string, timeout time.Duration, opt kconfig.Opt) error {
	dep, err := c.Api().AppsV1().Deployments(opt.Namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if kubeutil.DeploymentIsReady(dep) {
		return nil
	}

	wi, err := c.WatchDeploy(metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", name)}, opt)
	if err != nil {
		return err
	}

	timer := time.NewTimer(timeout)
	for {
		select {
		case <-timer.C:
			return except.NewError("Deploy failed to be ready after %s", except.ErrTimeout, timeout)
		case r := <-wi.ResultChan():
			switch r.Type {
			case watch.Error:
				reason := "unknown"
				if r.Object != nil {
					if dep, ok := r.Object.(*appsv1.Deployment); ok {
						if cond := c.getLatestCondition(dep); cond != nil {
							reason = cond.Message
						}
					}
				}
				return except.NewError("Deploy %s failed: %s", except.ErrInternalError, reason)
			case watch.Modified:
				if r.Object != nil {
					if dep, ok := r.Object.(*appsv1.Deployment); ok {
						if kubeutil.DeploymentIsReady(dep) {
							return nil
						}
					}
				}
			}
		}
	}
}

func (c *client) DeleteDeploy(name string, opt kconfig.Opt) error {
	return c.Api().AppsV1().Deployments(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *client) UpsertDeploy(dep *appsv1.Deployment, opt kconfig.Opt) (*appsv1.Deployment, error) {
	deploy, err := c.Api().AppsV1().Deployments(opt.Namespace).Create(dep)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return c.Api().AppsV1().Deployments(opt.Namespace).Update(dep)
		} else {
			return nil, err
		}
	}
	return deploy, nil
}

func (c *client) DeleteConfigMap(name string, opt kconfig.Opt) error {
	return c.Api().CoreV1().ConfigMaps(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *client) UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	cmApi := c.Api().CoreV1().ConfigMaps(opt.Namespace)

	out, err := cmApi.Create(cm)
	if errors.IsAlreadyExists(err) {
		out, err = cmApi.Update(cm)
	}

	return out, err
}

func (c *client) WatchPods(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error) {
	return c.Api().CoreV1().Pods(opt.Namespace).Watch(lo)
}

func (c *client) getLatestCondition(dep *appsv1.Deployment) *appsv1.DeploymentCondition {
	if len(dep.Status.Conditions) > 0 {
		return &dep.Status.Conditions[len(dep.Status.Conditions)-1]
	}
	return nil
}
