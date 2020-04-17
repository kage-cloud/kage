package store

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kage-cloud/kage/xds/pkg/model/consts"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type KubeStoreSpec struct {
	Interface kubernetes.Interface
	Namespace string
}

func NewKubeStore(spec *KubeStoreSpec) (EnvoyStatePersistentStore, error) {
	return &kubeStore{
		Interface: spec.Interface,
		Namespace: spec.Namespace,
	}, nil
}

type kubeStore struct {
	Interface kubernetes.Interface
	Namespace string
}

func (k *kubeStore) Delete(name string) error {
	return k.Interface.CoreV1().ConfigMaps(k.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (k *kubeStore) FetchAll() ([]EnvoyState, error) {
	lo := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", consts.LabelKeyResource, consts.LabelValueResourceSnapshot),
	}

	cms, err := k.Interface.CoreV1().ConfigMaps(k.Namespace).List(lo)
	if err != nil {
		return nil, err
	}

	states := make([]EnvoyState, len(cms.Items))

	for i, c := range cms.Items {
		es, err := k.configMapToEnvoyState(&c)
		if err != nil {
			return nil, err
		}
		states[i] = *es
	}

	return states, nil
}

func (k *kubeStore) Save(state *EnvoyState) (SaveHandler, error) {
	buf := bytes.NewBuffer([]byte{})
	err := new(jsonpb.Marshaler).Marshal(buf, state)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      state.NodeId,
			Namespace: k.Namespace,
			Labels: map[string]string{
				consts.LabelKeyDomain:   consts.Domain,
				consts.LabelKeyResource: consts.LabelValueResourceSnapshot,
			},
		},
		BinaryData: map[string][]byte{
			state.NodeId: buf.Bytes(),
		},
	}

	prevCm, err := k.Interface.CoreV1().ConfigMaps(k.Namespace).Get(state.NodeId, metav1.GetOptions{})
	if err != nil {
		log.WithField("name", state.NodeId).
			WithField("namespace", k.Namespace).
			Debug("No previous configmap found")
	}

	cm, err = upsertConfigMap(k.Interface, cm, k.Namespace)
	if err != nil {
		return nil, err
	}

	return NewKubeSaveHandler(prevCm, cm, k.Interface), nil
}

func (k *kubeStore) Fetch(name string) (*EnvoyState, error) {
	cm, err := k.Interface.CoreV1().ConfigMaps(k.Namespace).Get(name, metav1.GetOptions{})
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
	if err := jsonpb.Unmarshal(bytes.NewReader(cm.BinaryData[cm.Name]), es); err != nil {
		return nil, err
	}
	return es, nil
}

func upsertConfigMap(api kubernetes.Interface, cm *corev1.ConfigMap, namespace string) (*corev1.ConfigMap, error) {
	out, err := api.CoreV1().ConfigMaps(namespace).Create(cm)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return api.CoreV1().ConfigMaps(namespace).Update(cm)
		} else {
			return nil, err
		}
	}
	return out, nil
}

func NewKubeSaveHandler(prevCm, cm *corev1.ConfigMap, client kubernetes.Interface) SaveHandler {
	return &kubeSaveHandler{
		cm:            cm,
		prevConfigMap: prevCm,
		api:           client,
	}
}

type kubeSaveHandler struct {
	prevConfigMap *corev1.ConfigMap
	cm            *corev1.ConfigMap
	api           kubernetes.Interface
}

func (k *kubeSaveHandler) Revert() error {
	if k.prevConfigMap == nil {
		return k.api.CoreV1().ConfigMaps(k.cm.Namespace).Delete(k.cm.Name, nil)
	} else {
		_, err := upsertConfigMap(k.api, k.prevConfigMap, k.prevConfigMap.Namespace)
		return err
	}
}
