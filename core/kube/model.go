package kube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type Filter func(object metav1.Object) bool
