package controlplane

import (
	"context"
	"fmt"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	eds "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	lds "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	rds "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	serverv3 "github.com/envoyproxy/go-control-plane/pkg/server/v3"

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

type cb struct {
}

func (c cb) OnStreamOpen(ctx context.Context, i int64, s string) error {
	fmt.Println("on stream open!!!", i, s)
	return nil
}

func (c cb) OnStreamClosed(i int64) {
	fmt.Println("on stream closed!!!")
}

func (c cb) OnStreamRequest(i int64, request *discovery.DiscoveryRequest) error {
	fmt.Println("on stream req!!!")
	return nil
}

func (c cb) OnStreamResponse(i int64, request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
	fmt.Println("on stream resp!!!")
}

func (c cb) OnFetchRequest(ctx context.Context, request *discovery.DiscoveryRequest) error {
	fmt.Println("on fetch req!!!")
	return nil
}

func (c cb) OnFetchResponse(request *discovery.DiscoveryRequest, response *discovery.DiscoveryResponse) {
	fmt.Println("on fetch resp!!!")
}

func (e *envoyControlPlane) StartAsync() error {
	server := serverv3.NewServer(context.Background(), e.StoreClient.SnapshotCache(), cb{})

	grpcServer := grpc.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", e.Config.Xds.Port))
	if err != nil {
		return err
	}

	eds.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	rds.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	lds.RegisterListenerDiscoveryServiceServer(grpcServer, server)

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
