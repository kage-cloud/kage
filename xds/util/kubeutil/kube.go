package kubeutil

import (
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

// Resolves the namespace of the pod template for the deployment. if the namespace is listed on the pod template,
// returns that value. if the namespace is listed on the deployment, that value is used.
func DeploymentPodNamespace(deployment *appsv1.Deployment) string {
	ns := deployment.Spec.Template.Namespace
	if ns == "" {
		ns = deployment.Namespace
	}
	return ns
}

func ObjectKey(object runtime.Object, postfix ...string) string {
	objKind := object.GetObjectKind().GroupVersionKind()
	s := make([]string, 0)
	if o, ok := object.(metav1.Object); ok {
		s = append(s, fmt.Sprintf("%s-%s-%s", o.GetNamespace(), objKind.Kind, o.GetName()))
	}

	join := strings.Join(append(s, postfix...), "-")

	return join
}
