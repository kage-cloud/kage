package kubeinformer

import "context"

type Interface interface {
	Inform(ctx context.Context) error
}
