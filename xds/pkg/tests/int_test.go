package tests

import (
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"
	"time"
)

type IntTestSuite struct {
	K3sSuite
}

func (i *IntTestSuite) Test() {
	k8, err := kube.FromApiConfig(i.KubeConfig)
	if !i.NoError(err) {
		return
	}

	meta := metav1.ObjectMeta{
		Name:      "nginx",
		Namespace: "default",
		Labels: map[string]string{
			"app": "nginx",
		},
	}
	nginxDeploy := &appsv1.Deployment{
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: meta.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: meta.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "nginx",
						},
					},
				},
			},
		},
	}

	svcPort, _ := i.GetFreePort()
	nginxSvc := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector: meta.Labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "rpc",
					TargetPort: intstr.FromInt(80),
					Port:       int32(svcPort),
				},
			},
		},
	}

	nginxDeploy, err = i.Kube.AppsV1().Deployments(meta.Namespace).Create(nginxDeploy)
	if !i.NoError(err) {
		return
	}

	if err := k8.WaitTillDeployReady(nginxDeploy.Name, 1*time.Minute, kconfig.Opt{Namespace: meta.Namespace}); !i.NoError(err) {
		return
	}

	_, err = i.Kube.CoreV1().Services(meta.Namespace).Create(nginxSvc)
	if !i.NoError(err) {
		return
	}

}

func TestIntTestSuite(t *testing.T) {
	suite.Run(t, new(IntTestSuite))
}
