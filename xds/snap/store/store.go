package store

type EnvoyStateStore interface {
	Save(state *EnvoyState) (SaveHandler, error)
	Fetch(name string) (*EnvoyState, error)
	FetchAll() ([]EnvoyState, error)
}

type SaveHandler interface {
	Revert() error
}
