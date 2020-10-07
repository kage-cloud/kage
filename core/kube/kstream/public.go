package kstream

import (
	"github.com/kage-cloud/kage/core/kube/kfilter"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type ForEachFunc func(obj runtime.Object)

type MapFunc func(obj runtime.Object) runtime.Object

func StreamFromList(li metav1.ListInterface) Streamer {
	isController := false
	var objs []runtime.Object
	switch typ := li.(type) {
	case *corev1.PodList:
		isController = true
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *appsv1.DeploymentList:
		isController = true
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *appsv1.StatefulSetList:
		isController = true
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *appsv1.DaemonSetList:
		isController = true
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *appsv1.ReplicaSetList:
		isController = true
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *corev1.ConfigMapList:
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *corev1.ServiceList:
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *corev1.EndpointsList:
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *corev1.SecretList:
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	case *corev1.ServiceAccountList:
		objs = make([]runtime.Object, len(typ.Items))
		for i := range typ.Items {
			objs[i] = &typ.Items[i]
		}
	default:
		objs = make([]runtime.Object, 0)
	}

	return &streamer{
		li:           li,
		isController: isController,
		objs:         objs,
	}
}

type Streamer interface {
	IsController() bool

	First() runtime.Object

	Objects() []runtime.Object

	MetaObjects() []metav1.Object

	ListInterface() metav1.ListInterface

	// If the underlying object's type has a LabelSelector, return them. If the object is a Pod, a Selector which matches
	// that Pod is returned.
	LabelSelectors() []labels.Selector
	ForEach(forEachFunc ForEachFunc) Streamer
	Filter(filter kfilter.Filter) Streamer
	Map(mapper MapFunc) Streamer
	Len() int
}

type streamer struct {
	li           metav1.ListInterface
	isController bool
	objs         []runtime.Object
}

func (s *streamer) MetaObjects() []metav1.Object {
	metaObjs := make([]metav1.Object, 0, len(s.objs))

	for _, obj := range s.objs {
		if v, ok := obj.(metav1.Object); ok {
			metaObjs = append(metaObjs, v)
		}
	}

	return metaObjs
}

func (s *streamer) ListInterface() metav1.ListInterface {
	return ktypes.SetObjects(s.li, s.objs)
}

func (s *streamer) Filter(filter kfilter.Filter) Streamer {
	filtered := make([]runtime.Object, 0, len(s.objs))
	for _, v := range s.objs {
		if metaObj, ok := v.(metav1.Object); ok {
			if filter(metaObj) {
				filtered = append(filtered, v)
			}
		}
	}
	s.objs = filtered
	return s
}

func (s *streamer) Objects() []runtime.Object {
	return s.objs
}

func (s *streamer) First() runtime.Object {
	if len(s.objs) > 0 {
		return s.objs[0]
	} else {
		return nil
	}
}

func (s *streamer) LabelSelectors() []labels.Selector {
	selectors := make([]labels.Selector, 0, len(s.objs))

	for _, v := range s.objs {
		if v, ok := v.(*corev1.Pod); ok && v.Labels != nil {
			selectors = append(selectors, labels.SelectorFromValidatedSet(v.Labels))
			continue
		}
		ls := ktypes.GetLabelSelector(v)
		if ls != nil {
			selectors = append(selectors, ls)
		}
	}
	return selectors
}

func (s *streamer) Len() int {
	return len(s.objs)
}

func (s *streamer) IsController() bool {
	return s.isController
}

func (s *streamer) ForEach(forEachFunc ForEachFunc) Streamer {
	for _, v := range s.objs {
		forEachFunc(v)
	}
	return s
}

func (s *streamer) Map(mapper MapFunc) Streamer {
	for i, v := range s.objs {
		s.objs[i] = mapper(v)
	}
	return s
}
