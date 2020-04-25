package service

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kengine/objconv"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const KubeReaderServiceKey = "KubeReaderService"

type KubeReaderService interface {
	GetDeployReplicaSets(deploys []appsv1.Deployment, opt kconfig.Opt) ([]appsv1.ReplicaSet, error)
	GetReplicaSet(name string, opt kconfig.Opt) (*appsv1.ReplicaSet, error)
	GetServices(name string, opt kconfig.Opt) (*corev1.Service, error)
	GetPods(name string, opt kconfig.Opt) (*corev1.Pod, error)
	GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error)
	GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error)
	GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error)
	ListServices(selector labels.Selector, opt kconfig.Opt) ([]corev1.Service, error)
	ListPods(selector labels.Selector, opt kconfig.Opt) ([]corev1.Pod, error)
	ListDeploys(selector labels.Selector, opt kconfig.Opt) ([]appsv1.Deployment, error)
	ListConfigMaps(selector labels.Selector, opt kconfig.Opt) ([]corev1.ConfigMap, error)
	ListEndpoints(selector labels.Selector, opt kconfig.Opt) ([]corev1.Endpoints, error)
}

type kubeReaderService struct {
	KubeClient     kube.Client         `inject:"KubeClient"`
	InformerClient kube.InformerClient `inject:"InformerClient"`
}

func (k *kubeReaderService) GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindEndpoints)
	if k.InformerClient.Informing(nsKind) {
		obj, err := k.InformerClient.Get(nsKind, name)
		if err == nil {
			return obj.(*corev1.Endpoints), nil
		}
	}
	return k.KubeClient.Api().CoreV1().Endpoints(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) ListEndpoints(selector labels.Selector, opt kconfig.Opt) ([]corev1.Endpoints, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      ktypes.KindEndpoints,
	}
	if k.InformerClient.Informing(nsKind) {
		list, err := k.InformerClient.List(nsKind, selector)
		if err == nil {
			return list.(*corev1.EndpointsList).Items, nil
		}
	}

	list, err := k.KubeClient.Api().CoreV1().Endpoints(opt.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *kubeReaderService) GetServices(name string, opt kconfig.Opt) (*corev1.Service, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindConfigMap)
	if k.InformerClient.Informing(nsKind) {
		obj, err := k.InformerClient.Get(nsKind, name)
		if err == nil {
			return obj.(*corev1.Service), nil
		}
	}
	return k.KubeClient.Api().CoreV1().Services(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) GetPods(name string, opt kconfig.Opt) (*corev1.Pod, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindConfigMap)
	if k.InformerClient.Informing(nsKind) {
		obj, err := k.InformerClient.Get(nsKind, name)
		if err == nil {
			return obj.(*corev1.Pod), nil
		}
	}
	return k.KubeClient.Api().CoreV1().Pods(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindConfigMap)
	if k.InformerClient.Informing(nsKind) {
		obj, err := k.InformerClient.Get(nsKind, name)
		if err == nil {
			return obj.(*appsv1.Deployment), nil
		}
	}
	return k.KubeClient.Api().AppsV1().Deployments(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, ktypes.KindConfigMap)
	if k.InformerClient.Informing(nsKind) {
		obj, err := k.InformerClient.Get(nsKind, name)
		if err == nil {
			return obj.(*corev1.ConfigMap), nil
		}
	}
	return k.KubeClient.Api().CoreV1().ConfigMaps(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) ListPods(selector labels.Selector, opt kconfig.Opt) ([]corev1.Pod, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      ktypes.KindPod,
	}
	if k.InformerClient.Informing(nsKind) {
		list, err := k.InformerClient.List(nsKind, selector)
		if err == nil {
			return list.(*corev1.PodList).Items, nil
		}
	}

	list, err := k.KubeClient.Api().CoreV1().Pods(opt.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *kubeReaderService) ListDeploys(selector labels.Selector, opt kconfig.Opt) ([]appsv1.Deployment, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      ktypes.KindDeploy,
	}
	if k.InformerClient.Informing(nsKind) {
		list, err := k.InformerClient.List(nsKind, selector)
		if err == nil {
			return list.(*appsv1.DeploymentList).Items, nil
		}
	}

	list, err := k.KubeClient.Api().AppsV1().Deployments(opt.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *kubeReaderService) ListConfigMaps(selector labels.Selector, opt kconfig.Opt) ([]corev1.ConfigMap, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      ktypes.KindConfigMap,
	}
	if k.InformerClient.Informing(nsKind) {
		list, err := k.InformerClient.List(nsKind, selector)
		if err == nil {
			return list.(*corev1.ConfigMapList).Items, nil
		}
	}

	list, err := k.KubeClient.Api().CoreV1().ConfigMaps(opt.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (k *kubeReaderService) ListServices(selector labels.Selector, opt kconfig.Opt) ([]corev1.Service, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      ktypes.KindService,
	}
	if k.InformerClient.Informing(nsKind) {
		list, err := k.InformerClient.List(nsKind, selector)
		if err == nil {
			return list.(*corev1.ServiceList).Items, nil
		}
	}

	svcs, err := k.KubeClient.Api().CoreV1().Services(opt.Namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}
	return svcs.Items, nil
}

func (k *kubeReaderService) GetReplicaSet(name string, opt kconfig.Opt) (*appsv1.ReplicaSet, error) {
	return k.KubeClient.Api().AppsV1().ReplicaSets(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (k *kubeReaderService) GetDeployReplicaSets(deploys []appsv1.Deployment, opt kconfig.Opt) ([]appsv1.ReplicaSet, error) {
	rses, err := k.KubeClient.Api().AppsV1().ReplicaSets(opt.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	filteredSets := kubeutil.FilterObject(kubeutil.OwnerFilter(objconv.FromDeployments(deploys)...), objconv.FromReplicaSets(rses.Items)...)

	return objconv.ToReplicaSetsUnsafe(filteredSets), nil
}
