package controlplane

import (
	"context"
	"fmt"
	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/kage-cloud/kage/xds/pkg/config"
	"github.com/kage-cloud/kage/xds/pkg/snap"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"time"
)

const EnvoyControlPlaneKey = "EnvoyControlPlane"

type Envoy interface {
	StartAsync() error
}

type envoyControlPlane struct {
	StoreClient snap.StoreClient `inject:"StoreClient"`
	Config      *config.Config   `inject:"Config"`
}

func (e *envoyControlPlane) StartAsync() error {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(context.Background(), snapshotCache, nil)

	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", e.Config.Xds.Port))
	if err != nil {
		return err
	}

	apiv2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	apiv2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	apiv2.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	errChan := make(chan error)
	go func() {
		log.WithField("port", e.Config.Xds.Port).Info("Started control plane server.")
		err := grpcServer.Serve(lis)
		if err != nil {
			errChan <- err
		}
	}()

	timer := time.NewTimer(1 * time.Second)
	select {
	case <-timer.C:
		break
	case e := <-errChan:
		return e
	}

	return nil
}
