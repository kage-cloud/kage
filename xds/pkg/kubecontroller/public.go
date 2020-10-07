package kubecontroller

import "context"

type Interface interface {
	StartAsync(ctx context.Context) error
}
