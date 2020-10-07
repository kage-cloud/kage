package meta

type Canary struct {
	Source            ObjRef `json:"source"`
	RoutingPercentage uint32 `json:"routing_percentage"`
}

func (c *Canary) GetDomain() string {
	return DomainCanary
}

type ObjRef struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}
