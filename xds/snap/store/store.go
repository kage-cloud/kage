package store

import (
	"encoding/json"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

const (
	LabelValueDomainKage       = "com.eddieowens.kage"
	LabelValueResourceSnapshot = "snapshot"
)

const (
	LabelKeyDomain   = "domain"
	LabelKeyResource = LabelValueDomainKage + "/resource"
)

type EnvoyStateStore interface {
	Save(state *EnvoyState) (SaveHandler, error)
	Fetch(name string) (*EnvoyState, error)
}

type SaveHandler interface {
	Revert() error
}

func NewKubeStore() (EnvoyStateStore, error) {
	kubeClient, err := kube.NewClient()
	if err != nil {
		return nil, err
	}

	return &kubeStore{
		KubeClient: kubeClient,
	}, nil
}

type kubeStore struct {
	KubeClient kube.Client
}

func (k *kubeStore) Save(state *EnvoyState) (SaveHandler, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	namespace := os.Getenv("NAMESPACE")

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      state.Name,
			Namespace: namespace,
			Labels: map[string]string{
				LabelKeyDomain:   LabelValueDomainKage,
				LabelKeyResource: LabelValueResourceSnapshot,
			},
		},
		BinaryData: map[string][]byte{
			state.Name: b,
		},
	}

	opt := kconfig.Opt{Namespace: namespace}

	cm, err = k.KubeClient.UpsertConfigMap(cm, opt)
	if err != nil {
		return nil, err
	}

	return NewKubeSaveHandler(cm, k.KubeClient), nil
}

func (k *kubeStore) Fetch(name string) (*EnvoyState, error) {
	namespace := os.Getenv("NAMESPACE")

	cm, err := k.KubeClient.GetConfigMap(name, kconfig.Opt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	es := new(EnvoyState)
	if err := json.Unmarshal(cm.BinaryData[cm.Name], es); err != nil {
		return nil, err
	}

	return es, nil
}

func NewKubeSaveHandler(cm *corev1.ConfigMap, client kube.Client) SaveHandler {
	return &kubeSaveHandler{
		prevConfigMap: cm,
		client:        client,
	}
}

type kubeSaveHandler struct {
	prevConfigMap *corev1.ConfigMap
	client        kube.Client
}

func (k *kubeSaveHandler) Revert() error {
	return k.client.DeleteConfigMap(k.prevConfigMap.Name, kconfig.Opt{Namespace: k.prevConfigMap.Namespace})
}
