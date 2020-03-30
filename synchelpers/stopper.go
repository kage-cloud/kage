package synchelpers

type Stopper interface {
	Stop(err error)
	IsStopped() bool
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
	onStop    OnStopFunc
	isStopped bool
}

func (s *stopper) IsStopped() bool {
	return s.isStopped
}

func (s *stopper) Stop(err error) {
	s.isStopped = true
	if s.onStop != nil {
		s.onStop(err)
	}
}
