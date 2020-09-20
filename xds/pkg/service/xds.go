package service

import (
	"context"
	"github.com/kage-cloud/kage/core/except"
	"github.com/kage-cloud/kage/core/kube/kconfig"
	"github.com/kage-cloud/kage/core/kube/kubeutil"
	"github.com/kage-cloud/kage/xds/pkg/factory"
	"github.com/kage-cloud/kage/xds/pkg/model"
	"github.com/kage-cloud/kage/xds/pkg/model/envoyepctlr"
	"github.com/kage-cloud/kage/xds/pkg/model/xds"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	"github.com/kage-cloud/kage/xds/pkg/snap/store"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const XdsServiceKey = "XdsService"

type XdsService interface {
	StartControlPlane(ctx context.Context, spec *xds.ControlPlaneSpec) error
	StopControlPlane(nodeId string) error
	SetRoutingWeight(meshConfig *model.MeshConfig) error
}

type xdsService struct {
	EnvoyEndpointController EnvoyEndpointController `inject:"EnvoyEndpointController"`
	WatchService            WatchService            `inject:"WatchService"`
	RouteFactory            factory.RouteFactory    `inject:"RouteFactory"`
	StoreClient             snap.StoreClient        `inject:"StoreClient"`
}

func (x *xdsService) StopControlPlane(nodeId string) error {
	if err := x.StoreClient.Delete(nodeId); err != nil {
		return err
	}

	return nil
}

func (x *xdsService) SetRoutingWeight(meshConfig *model.MeshConfig) error {
	routes := x.RouteFactory.FromPercentage(meshConfig)

	state := &store.EnvoyState{
		NodeId: meshConfig.NodeId,
		Routes: routes,
	}

	return x.StoreClient.Set(state)
}

func (x *xdsService) StartControlPlane(ctx context.Context, spec *xds.ControlPlaneSpec) error {
	if x.controlPlaneExists(spec.MeshConfig.NodeId) {
		return except.NewError("The control plane for canary %s already exists", except.ErrAlreadyExists, spec.CanaryDeploy.Name)
	}

	if err := x.SetRoutingWeight(&spec.MeshConfig); err != nil {
		return err
	}

	canSelector, err := metav1.LabelSelectorAsSelector(spec.CanaryDeploy.Spec.Selector)
	if err != nil {
		return err
	}

	targetSelector, err := metav1.LabelSelectorAsSelector(spec.TargetDeploy.Spec.Selector)
	if err != nil {
		return err
	}

	ns := kubeutil.DeploymentPodNamespace(spec.TargetDeploy)
	opt := kconfig.Opt{Namespace: ns}

	eepcSpec := &envoyepctlr.Spec{
		NodeId: spec.MeshConfig.NodeId,
		Selectors: []labels.Selector{
			canSelector,
			targetSelector,
		},
		Opt: opt,
	}

	if err := x.EnvoyEndpointController.StartAsync(ctx, eepcSpec); err != nil {
		return err
	}

	log.WithField("name", spec.CanaryDeploy.Name).
		WithField("namespace", spec.CanaryDeploy.Namespace).
		WithField("node_id", spec.MeshConfig.NodeId).
		Debug("Added canary to control plane.")

	return nil
}

func (x *xdsService) controlPlaneExists(nodeId string) bool {
	_, err := x.StoreClient.Get(nodeId)
	exists := err == nil
	if !exists {
		log.WithField("node_id", nodeId).WithError(err).Debug("Control plane doesn't exist")
	}
	return exists
}
