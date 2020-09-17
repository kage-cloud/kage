package tests

import (
	"context"
	"fmt"
	"github.com/rancher/k3d/v3/pkg/cluster"
	"github.com/rancher/k3d/v3/pkg/runtimes"
	"github.com/rancher/k3d/v3/pkg/types"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"net"
)

type K3sSuite struct {
	suite.Suite
	Runtime    runtimes.Runtime
	Cluster    *types.Cluster
	Kube       kubernetes.Interface
	KubeConfig *api.Config
}

func (k *K3sSuite) TearDownSuite() {
	_ = cluster.ClusterDelete(context.Background(), k.Runtime, k.Cluster)
}

func (k *K3sSuite) SetupSuite() {
	if k.Runtime == nil {
		k.Runtime = runtimes.Docker
	}

	exposedPort, err := k.GetFreePort()
	if err != nil {
		k.FailNow(err.Error())
	}
	k.Cluster = &types.Cluster{
		Name: types.DefaultClusterName,
		ServerLoadBalancer: &types.Node{
			Role:  types.LoadBalancerRole,
			Image: "rancher/k3d-proxy:v3.0.1",
		},
		Nodes: make([]*types.Node, 0, 1),
		CreateClusterOpts: &types.ClusterCreateOpts{
			WaitForServer: true,
		},
		ExposeAPI: types.ExposeAPI{
			Port:   fmt.Sprintf("%d", exposedPort),
			Host:   types.DefaultAPIHost,
			HostIP: types.DefaultAPIHost,
		},
	}

	node := &types.Node{
		Role:  types.ServerRole,
		Image: "docker.io/rancher/k3s",
		Args:  k.Cluster.CreateClusterOpts.K3sServerArgs,
	}

	k.Cluster.Nodes = append(k.Cluster.Nodes, node, k.Cluster.ServerLoadBalancer)

	ctx, _ := context.WithCancel(context.Background())
	rt := runtimes.Docker

	if err := cluster.ClusterCreate(ctx, rt, k.Cluster); err != nil {
		k.FailNow(err.Error())
	}

	k.KubeConfig, err = cluster.KubeconfigGet(ctx, rt, k.Cluster)
	if err != nil {
		k.FailNow(err.Error())
	}

	conf, err := clientcmd.NewDefaultClientConfig(
		*k.KubeConfig,
		&clientcmd.ConfigOverrides{},
	).ClientConfig()

	if err != nil {
		k.FailNow(err.Error())
	}

	k.Kube, err = kubernetes.NewForConfig(conf)
	if err != nil {
		k.FailNow(err.Error())
	}
}

func (k *K3sSuite) GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
