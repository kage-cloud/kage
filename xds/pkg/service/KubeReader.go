package service

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/core/kube/objconv"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const KubeReaderServiceKey = "KubeReaderService"

type KubeReaderService interface {
	GetDeployReplicaSets(deploys []appsv1.Deployment, opt kconfig.Opt) ([]appsv1.ReplicaSet, error)
	GetReplicaSet(name string, opt kconfig.Opt) (*appsv1.ReplicaSet, error)
}

type kubeReaderService struct {
	KubeClient kube.Client `inject:"KubeClient"`
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
