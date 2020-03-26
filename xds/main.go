package main

import (
	"context"
	"fmt"
	"github.com/kage-cloud/kage/xds/snap"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"google.golang.org/grpc"
	"net"
	"os"
)

func main() {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(context.Background(), snapshotCache, nil)

	grpcServer := grpc.NewServer()

	xdsPort := os.Getenv("KAGE_XDS_PORT")

	lis, _ := net.Listen("tcp", ":"+xdsPort)

	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	var err error
	SnapClient, err = snap.NewStoreClient()
	if err != nil {
		panic(err)
	}

	fmt.Println(grpcServer.Serve(lis))
}
