package store

type Filter func(d interface{}) bool

type EnvoyStatePersistentStore interface {
	Save(state *EnvoyState) (SaveHandler, error)
	Fetch(nodeId string) (*EnvoyState, error)
	FetchAll() ([]EnvoyState, error)
	Delete(nodeId string) error
}

type SaveHandler interface {
	Revert() error
}
