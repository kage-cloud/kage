package meta

func ProxyDomain(svcName string) string {
	return svcName + "." + DomainXds
}

type ServiceProxy struct {
	Name     string            `json:"name"`
	Selector map[string]string `json:"selector"`
}

type ProxiedService struct {
	Selector map[string]string `json:"selector"`
}

func (p *ProxiedService) GetDomain() string {
	return DomainXds
}
