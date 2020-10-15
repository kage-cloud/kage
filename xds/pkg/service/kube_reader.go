package service

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/core/kube/ktypes/objconv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

const KubeReaderServiceKey = "KubeReaderService"

type KubeReaderService interface {
	Get(name string, kind ktypes.Kind, opt kconfig.Opt) (runtime.Object, error)
	GetDeployReplicaSets(deploys []appsv1.Deployment, opt kconfig.Opt) ([]appsv1.ReplicaSet, error)
	GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error)
	GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error)
	GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error)
	GetController(obj metav1.Object, opt kconfig.Opt) (runtime.Object, error)
	WalkControllers(obj metav1.Object, walker ktypes.ControllerWalker) error

	List(selector labels.Selector, kind ktypes.Kind, opt kconfig.Opt) (metav1.ListInterface, error)

	// Find all resources that select the specified kind. E.g. List all services whose selectors match the provided set.
	ListSelected(set labels.Set, kind ktypes.Kind, opt kconfig.Opt) (metav1.ListInterface, error)
	ListServices(selector labels.Selector, opt kconfig.Opt) ([]corev1.Service, error)
	ListPods(selector labels.Selector, opt kconfig.Opt) ([]corev1.Pod, error)
	ListDeploys(selector labels.Selector, opt kconfig.Opt) ([]appsv1.Deployment, error)
}

type kubeReaderService struct {
	KubeClient     kube.Client         `inject:"KubeClient"`
	InformerClient kube.InformerClient `inject:"InformerClient"`
}

func (k *kubeReaderService) ListSelected(set labels.Set, kind ktypes.Kind, opt kconfig.Opt) (metav1.ListInterface, error) {
	li, err := k.List(labels.Everything(), kind, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(li).Filter(kfilter.SelectsSet(set)).Collect().ListInterface(), nil
}

func (k *kubeReaderService) WalkControllers(obj metav1.Object, walker ktypes.ControllerWalker) error {
	if len(obj.GetOwnerReferences()) == 0 {
		_, err := walker(obj.(runtime.Object))
		return err
	}

	opt := kconfig.Opt{Namespace: obj.GetNamespace()}

	for _, v := range obj.GetOwnerReferences() {
		if v.Controller != nil && *v.Controller {
			obj, err := k.Get(v.Name, ktypes.Kind(v.Kind), opt)
			if err != nil {
				return err
			}

			if keepWalking, err := walker(obj); !keepWalking || err != nil {
				return err
			}

			if metaObj, ok := obj.(metav1.Object); ok {
				return k.WalkControllers(metaObj, walker)
			}
		}
	}

	return nil
}

func (k *kubeReaderService) List(selector labels.Selector, kind ktypes.Kind, opt kconfig.Opt) (metav1.ListInterface, error) {
	nsKind := ktypes.NamespaceKind{
		Namespace: opt.Namespace,
		Kind:      kind,
	}
	if k.InformerClient.Informing(nsKind) {
		return k.InformerClient.List(nsKind, selector)
	}

	return k.KubeClient.List(kind, metav1.ListOptions{LabelSelector: selector.String()}, opt)
}

func (k *kubeReaderService) Get(name string, kind ktypes.Kind, opt kconfig.Opt) (runtime.Object, error) {
	nsKind := ktypes.NewNamespaceKind(opt.Namespace, kind)
	if k.InformerClient.Informing(nsKind) {
		return k.InformerClient.Get(nsKind, name)
	}
	return k.KubeClient.Get(name, kind, opt)
}

func (k *kubeReaderService) GetController(obj metav1.Object, opt kconfig.Opt) (runtime.Object, error) {
	for _, v := range obj.GetOwnerReferences() {
		if v.Controller != nil && *v.Controller {
			obj, err := k.Get(v.Name, ktypes.Kind(v.Kind), opt)
			if err != nil {
				return nil, err
			}
			if metaObj, ok := obj.(metav1.Object); ok {
				return k.GetController(metaObj, opt)
			}
		}
	}

	return obj.(runtime.Object), nil
}

func (k *kubeReaderService) GetEndpoints(name string, opt kconfig.Opt) (*corev1.Endpoints, error) {
	obj, err := k.Get(name, ktypes.KindEndpoints, opt)
	if err != nil {
		return nil, err
	}
	return obj.(*corev1.Endpoints), nil
}

func (k *kubeReaderService) ListEndpoints(selector labels.Selector, opt kconfig.Opt) ([]corev1.Endpoints, error) {
	li, err := k.List(selector, ktypes.KindEndpoints, opt)
	if err != nil {
		return nil, err
	}

	return li.(*corev1.EndpointsList).Items, nil
}

func (k *kubeReaderService) GetDeploy(name string, opt kconfig.Opt) (*appsv1.Deployment, error) {
	obj, err := k.Get(name, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}
	return obj.(*appsv1.Deployment), nil
}

func (k *kubeReaderService) GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	obj, err := k.Get(name, ktypes.KindConfigMap, opt)
	if err != nil {
		return nil, err
	}
	return obj.(*corev1.ConfigMap), nil
}

func (k *kubeReaderService) ListPods(selector labels.Selector, opt kconfig.Opt) ([]corev1.Pod, error) {
	li, err := k.List(selector, ktypes.KindPod, opt)
	if err != nil {
		return nil, err
	}

	return li.(*corev1.PodList).Items, nil
}

func (k *kubeReaderService) ListDeploys(selector labels.Selector, opt kconfig.Opt) ([]appsv1.Deployment, error) {
	li, err := k.List(selector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return li.(*appsv1.DeploymentList).Items, nil
}

func (k *kubeReaderService) ListServices(selector labels.Selector, opt kconfig.Opt) ([]corev1.Service, error) {
	li, err := k.List(selector, ktypes.KindService, opt)
	if err != nil {
		return nil, err
	}

	return li.(*corev1.ServiceList).Items, nil
}

func (k *kubeReaderService) GetDeployReplicaSets(deploys []appsv1.Deployment, opt kconfig.Opt) ([]appsv1.ReplicaSet, error) {
	li, err := k.List(labels.Everything(), ktypes.KindReplicaSet, opt)
	if err != nil {
		return nil, err
	}
	rses := li.(*appsv1.ReplicaSetList)

	filteredSets := kfilter.FilterObject(kfilter.OwnerFilter(objconv.FromDeployments(deploys)...), objconv.FromReplicaSets(rses.Items)...)

	return objconv.ToReplicaSetsUnsafe(filteredSets), nil
}
