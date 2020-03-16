package kubeutil

import appsv1 "k8s.io/api/apps/v1"

// Resolves the namespace of the pod template for the deployment. if the namespace is listed on the pod template,
// returns that value. if the namespace is listed on the deployment, that value is used.
func DeploymentPodNamespace(deployment *appsv1.Deployment) string {
	ns := deployment.Spec.Template.Namespace
	if ns == "" {
		ns = deployment.Namespace
	}
	return ns
}
