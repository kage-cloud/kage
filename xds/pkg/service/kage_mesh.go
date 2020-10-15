package service

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kstream"
	"github.com/kage-cloud/kage/core/kube/ktypes"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/meta"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

const KageMeshServiceKey = "KageMeshService"

var KageProxySelector = labels.SelectorFromValidatedSet(meta.ToMap(&meta.MeshMarker{IsMesh: true}))

type KageMeshService interface {
	CreateForCanary(canary *meta.Canary) (*meta.Xds, error)
	FetchForCanary(canary *meta.Canary) (*meta.Xds, error)

	MarshalXdsMeta(obj metav1.Object, xds *meta.Xds)
	UnmarshalXdsMeta(obj metav1.Object) (*meta.Xds, error)
	TargetsPod(xdsAnno *meta.Xds, pod *corev1.Pod) bool

	Remove(xds *meta.Xds, opt kconfig.Opt) error
	ListXdsForPod(pod *corev1.Pod) ([]meta.Xds, error)

	// TODO: make sure to handle service removal from the service selector. should we sync all services??
}

type kageMeshService struct {
	KubeClient        kube.Client             `inject:"KubeClient"`
	KubeReaderService KubeReaderService       `inject:"KubeReaderService"`
	KageMeshFactory   factory.KageMeshFactory `inject:"KageMeshFactory"`
	MeshConfigService MeshConfigService       `inject:"MeshConfigService"`
	ProxyService      ProxyService            `inject:"ProxyService"`
	StoreClient       snap.StoreClient        `inject:"StoreClient"`
}

func (k *kageMeshService) initServiceSelectors(ref meta.ObjRef) (map[string]map[string]string, error) {
	obj, err := k.fetchObjRefObj(ref)
	if err != nil {
		return nil, err
	}

	canaryMetaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, except.NewError("the canary object is not a valid kube meta object", except.ErrInvalid)
	}

	opt := kconfig.Opt{Namespace: canaryMetaObj.GetNamespace()}

	svcObjs, err := k.KubeReaderService.ListSelected(canaryMetaObj.GetLabels(), ktypes.KindService, opt)
	if err != nil {
		return nil, err
	}

	svcSelectors := map[string]map[string]string{}
	svcs := kstream.StreamFromList(svcObjs).Collect().Services()
	for _, svc := range svcs.Items {
		svcSelectors[svc.Name] = svc.Spec.Selector
	}

	return svcSelectors, nil
}

func (k *kageMeshService) FetchForCanary(canary *meta.Canary) (*meta.Xds, error) {
	opt := kconfig.Opt{Namespace: canary.CanaryObj.Namespace}

	name := k.genKageMeshName(&canary.SourceObj)
	dep, err := k.KubeReaderService.GetDeploy(name, opt)
	if err != nil {
		return nil, err
	}

	return k.UnmarshalXdsMeta(dep)
}

func (k *kageMeshService) Remove(xds *meta.Xds, opt kconfig.Opt) error {
	dep, err := k.KubeReaderService.GetDeploy(xds.Name, opt)
	if err != nil {
		return err
	}

	proxiedSvcs, err := k.listProxiedServicesForDeploy(dep)
	if err != nil {
		return err
	}

	for _, v := range proxiedSvcs.Items {
		if err := k.removeForService(&v, opt); err != nil {
			logrus.WithError(err).
				WithField("name", v.Name).
				WithField("namespace", v.Namespace).
				Error("Failed to unlock service.")
			return err
		}
	}

	return k.StoreClient.Delete(xds.Config.NodeId)
}

func (k *kageMeshService) ListXdsForPod(pod *corev1.Pod) ([]meta.Xds, error) {
	opt := kconfig.Opt{Namespace: pod.Namespace}
	meshes, err := k.listMeshDeploys(opt)
	if err != nil {
		return nil, err
	}

	return k.listXdsAnnosForPod(meshes, pod)
}

func (k *kageMeshService) CreateForCanary(canary *meta.Canary) (*meta.Xds, error) {
	kageMeshName := k.genKageMeshName(&canary.SourceObj)
	opt := kconfig.Opt{Namespace: canary.SourceObj.Namespace}
	kageMeshDeploy, err := k.KubeReaderService.GetDeploy(kageMeshName, opt)
	var xdsAnno *meta.Xds
	if errors.IsNotFound(err) {
		kageMeshDeploy, xdsAnno, err = k.createKageMeshDeploy(kageMeshName, canary, opt)
		if err != nil {
			return nil, err
		}
	} else {
		xdsAnno, err = k.UnmarshalXdsMeta(kageMeshDeploy)
		if err != nil {
			return nil, err
		}
	}

	return xdsAnno, nil
}

func (k *kageMeshService) removeForService(svc *corev1.Service, opt kconfig.Opt) error {
	meshes, err := k.listDeploysForProxiedService(svc)
	if err != nil {
		return err
	}

	if len(meshes.Items) <= 1 {
		logrus.WithField("name", svc.Name).
			WithField("namespace", svc.Namespace).
			Info("Stopping all proxies for service.")
		if err := k.ProxyService.ReleaseService(svc, opt); err != nil {
			return err
		}
	}
	return nil
}

func (k *kageMeshService) listDeploysForProxiedService(svc *corev1.Service) (*appsv1.DeploymentList, error) {
	opt := kconfig.Opt{Namespace: svc.Namespace}
	meshes, err := k.KubeReaderService.List(KageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(meshes).Filter(func(object metav1.Object) bool {
		xdsAnno, err := k.UnmarshalXdsMeta(object)
		if err != nil {
			return false
		}

		_, ok := xdsAnno.ServiceSelectors[svc.Name]
		return ok
	}).Collect().Deployments(), nil
}

func (k *kageMeshService) listProxiedServicesForDeploy(dep *appsv1.Deployment) (*corev1.ServiceList, error) {
	opt := kconfig.Opt{Namespace: dep.Namespace}
	xdsAnno := new(meta.Xds)
	if err := meta.FromMap(dep.Annotations, xdsAnno); err != nil {
		return nil, err
	}

	svcs := make([]corev1.Service, 0, len(xdsAnno.ServiceSelectors))
	for name := range xdsAnno.ServiceSelectors {
		obj, err := k.KubeReaderService.Get(name, ktypes.KindService, opt)
		if err != nil {
			continue
		}

		svcs = append(svcs, *obj.(*corev1.Service))
	}

	return &corev1.ServiceList{Items: svcs}, nil
}

func (k *kageMeshService) listMeshDeploys(opt kconfig.Opt) (*appsv1.DeploymentList, error) {
	meshes, err := k.KubeReaderService.List(KageProxySelector, ktypes.KindDeployment, opt)
	if err != nil {
		return nil, err
	}

	return kstream.StreamFromList(meshes).Collect().Deployments(), nil
}

func (k *kageMeshService) listXdsAnnosForPod(meshes *appsv1.DeploymentList, pod *corev1.Pod) ([]meta.Xds, error) {
	xdsAnnos := make([]meta.Xds, 0, len(meshes.Items))

	for _, v := range meshes.Items {
		xdsAnno, err := k.UnmarshalXdsMeta(&v)
		if err != nil {
			continue
		}

		if k.TargetsPod(xdsAnno, pod) {
			xdsAnnos = append(xdsAnnos, *xdsAnno)
		}
	}

	return xdsAnnos, nil
}

func (k *kageMeshService) TargetsPod(xdsAnno *meta.Xds, pod *corev1.Pod) bool {
	podSet := labels.Set(pod.Labels)

	for _, v := range xdsAnno.ServiceSelectors {
		if labels.SelectorFromValidatedSet(v).Matches(podSet) {
			return true
		}
	}
	return false
}

func (k *kageMeshService) createKageMeshDeploy(name string, canary *meta.Canary, opt kconfig.Opt) (*appsv1.Deployment, *meta.Xds, error) {
	xdsAnno := &meta.Xds{
		Name:   name,
		Canary: *canary,
		Config: meta.XdsConfig{
			XdsId: meta.XdsId{NodeId: uuid.New().String()},
			Canary: meta.EnvoyConfig{
				ClusterName: "canary",
			},
			Source: meta.EnvoyConfig{
				ClusterName: "source",
			},
		},
	}

	sourceSvcSelectors, err := k.initServiceSelectors(canary.SourceObj)
	if err != nil {
		return nil, nil, err
	}

	canarySvcSelectors, err := k.initServiceSelectors(canary.CanaryObj)
	if err != nil {
		return nil, nil, err
	}

	for k, v := range sourceSvcSelectors {
		canarySvcSelectors[k] = v
	}

	xdsAnno.ServiceSelectors = canarySvcSelectors

	baseline, err := k.MeshConfigService.FromXdsConfig(&xdsAnno.Config)
	if err != nil {
		return nil, nil, err
	}

	cm := k.KageMeshFactory.BaselineConfigMap(name, baseline)
	_, err = k.KubeClient.Create(cm, opt)
	if err != nil {
		return nil, nil, err
	}

	dep := k.KageMeshFactory.Deploy(name, &xdsAnno.Config)
	dep, err = k.KubeClient.CreateDeploy(dep, opt)
	if err != nil {
		return nil, nil, err
	}

	return dep, xdsAnno, nil
}

func (k *kageMeshService) MarshalXdsMeta(obj metav1.Object, xds *meta.Xds) {
	obj.SetAnnotations(meta.Merge(obj.GetAnnotations(), xds))
	obj.SetLabels(meta.Merge(obj.GetLabels(), &xds.Config.XdsId))
}

func (k *kageMeshService) UnmarshalXdsMeta(obj metav1.Object) (*meta.Xds, error) {
	xdsAnno := new(meta.Xds)
	if err := meta.FromMap(obj.GetAnnotations(), xdsAnno); err != nil {
		return nil, err
	}

	if err := meta.FromMap(obj.GetLabels(), xdsAnno); err != nil {
		return nil, err
	}

	return xdsAnno, nil
}

func (k *kageMeshService) fetchObjRefObj(o meta.ObjRef) (runtime.Object, error) {
	return k.KubeReaderService.Get(o.Name, ktypes.Kind(o.Kind), kconfig.Opt{Namespace: o.Namespace})
}

func (k *kageMeshService) genKageMeshName(source *meta.ObjRef) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-kage-mesh", source.Name, source.Kind))
}
