package kubecontroller

import "context"

const EndpointsKubeControllerKey = "EndpointsKubeController"

type Endpoints struct {
}

func (e *Endpoints) StartAsync(ctx context.Context) error {
	panic("implement me")
}
