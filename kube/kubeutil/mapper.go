package kubeutil

import corev1 "k8s.io/api/core/v1"

func PodToEndpointAddress(pod *corev1.Pod) *corev1.EndpointAddress {
	return &corev1.EndpointAddress{
		IP:       pod.Status.PodIP,
		NodeName: &pod.Spec.NodeName,
		TargetRef: &corev1.ObjectReference{
			Kind:            "Pod",
			Namespace:       pod.ObjectMeta.Namespace,
			Name:            pod.ObjectMeta.Name,
			UID:             pod.ObjectMeta.UID,
			ResourceVersion: pod.ObjectMeta.ResourceVersion,
		}}
}

func EndpointPortFromServicePort(servicePort *corev1.ServicePort, portNum int) *corev1.EndpointPort {
	return &corev1.EndpointPort{
		Name:     servicePort.Name,
		Port:     int32(portNum),
		Protocol: servicePort.Protocol,
	}
}
