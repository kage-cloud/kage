package meta

type CanaryMarker struct {
	Canary bool `json:"canary"`
}

func (k *CanaryMarker) GetDomain() string {
	return DomainBase
}
