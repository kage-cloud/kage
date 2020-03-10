package store

import (
	"encoding/json"
	"fmt"
	"github.com/eddieowens/kage/kube"
	"github.com/eddieowens/kage/kube/kconfig"
	"github.com/eddieowens/kage/xds/model/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

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

func (k *kubeStore) Delete(name string) error {
	namespace := os.Getenv("NAMESPACE")

	return k.KubeClient.DeleteConfigMap(name, kconfig.Opt{Namespace: namespace})
}

func (k *kubeStore) FetchAll() ([]EnvoyState, error) {
	namespace := os.Getenv("NAMESPACE")

	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", consts.LabelKeyResource, consts.LabelValueResourceSnapshot),
	}

	cms, err := k.KubeClient.ListConfigMaps(lo, kconfig.Opt{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	states := make([]EnvoyState, len(cms))

	for i, c := range cms {
		es, err := k.configMapToEnvoyState(&c)
		if err != nil {
			return nil, err
		}
		states[i] = *es
	}

	return states, nil
}

func (k *kubeStore) Save(state *EnvoyState) (SaveHandler, error) {
	b, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	namespace := os.Getenv("NAMESPACE")

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      state.Name,
			Namespace: namespace,
			Labels: map[string]string{
				consts.LabelKeyDomain:   consts.Domain,
				consts.LabelKeyResource: consts.LabelValueResourceSnapshot,
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

	es, err := k.configMapToEnvoyState(cm)
	if err != nil {
		return nil, err
	}

	return es, nil
}

func (k *kubeStore) configMapToEnvoyState(cm *corev1.ConfigMap) (*EnvoyState, error) {
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
