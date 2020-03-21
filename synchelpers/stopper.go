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

func NewErrChanStopper(stopFunc OnStopFunc) (Stopper, chan error) {
	errChan := make(chan error)
	stop := NewStopper(func(err error) {
		if stopFunc != nil {
			stopFunc(err)
		}
		errChan <- err
	})
	return stop, errChan
}

type stopper struct {
	onStop OnStopFunc
}

func (s stopper) Stop(err error) {
	if s.onStop != nil {
		s.onStop(err)
	}
}
