package synchelpers

type Stopper interface {
	Stop(err error)
}

type OnStopFunc func(err error)

func NewStopper(onStop OnStopFunc) Stopper {
	return &stopper{
		onStop: onStop,
	}
}

type stopper struct {
	onStop OnStopFunc
}

func (s stopper) Stop(err error) {
	if s.onStop != nil {
		s.onStop(err)
	}
}
