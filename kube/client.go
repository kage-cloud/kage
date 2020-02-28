package kube

import (
	"github.com/eddieowens/kage/kube/kconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type Client interface {
	WatchPods(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error)
	UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error)
	GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error)
	DeleteConfigMap(name string, opt kconfig.Opt) error
}

func NewClient() (Client, error) {
	conf, err := kconfig.NewConfigClient()
	if err != nil {
		return nil, err
	}

	return &client{
		Config: conf,
	}, nil
}

type client struct {
	Config kconfig.Config
}

func (c *client) DeleteConfigMap(name string, opt kconfig.Opt) error {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return err
	}

	return api.CoreV1().ConfigMaps(opt.Namespace).Delete(name, &metav1.DeleteOptions{})
}

func (c *client) UpsertConfigMap(cm *corev1.ConfigMap, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}

	cmApi := api.CoreV1().ConfigMaps(opt.Namespace)

	out, err := cmApi.Create(cm)
	if errors.IsAlreadyExists(err) {
		out, err = cmApi.Update(cm)
	}

	return out, err
}

func (c *client) GetConfigMap(name string, opt kconfig.Opt) (*corev1.ConfigMap, error) {
	api, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}

	return api.CoreV1().ConfigMaps(opt.Namespace).Get(name, metav1.GetOptions{})
}

func (c *client) WatchPods(lo metav1.ListOptions, opt kconfig.Opt) (watch.Interface, error) {
	inter, err := c.Config.Api(opt.Context)
	if err != nil {
		return nil, err
	}
	return inter.CoreV1().Pods(opt.Namespace).Watch(lo)
}
