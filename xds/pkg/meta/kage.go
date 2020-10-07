package meta

type Kage struct {
	Canary bool `json:"canary"`
}

func (k *Kage) GetDomain() string {
	return DomainBase
}
