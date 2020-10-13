package meta

type Proxy struct {
	ProxyMarker
	DeletedSelector map[string]string `json:"deleted_selector"`
}

func (l *Proxy) GetDomain() string {
	return DomainXds
}

type ProxyMarker struct {
	Proxied bool `json:"proxied"`
}

func (l *ProxyMarker) GetDomain() string {
	return DomainXds
}
