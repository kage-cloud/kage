package meta

type ControllerType string

const (
	SourceControllerType ControllerType = "source"
	CanaryControllerType ControllerType = "canary"
)

type MeshMarker struct {
	IsMesh bool `json:"is_mesh"`
}

func (m *MeshMarker) GetDomain() string {
	return DomainXds
}

type Xds struct {
	Name             string                       `json:"name"`
	ServiceSelectors map[string]map[string]string `json:"service_selectors"`
	Canary           Canary                       `json:"canary"`
	Config           XdsConfig                    `json:"config"`
}

func (x *Xds) GetDomain() string {
	return DomainXds
}

type XdsId struct {
	NodeId string `json:"node_id"`
}

func (x *XdsId) GetDomain() string {
	return DomainXds
}

type XdsConfig struct {
	XdsId
	Canary EnvoyConfig `json:"canary"`
	Source EnvoyConfig `json:"source"`
}

func (x *XdsConfig) GetDomain() string {
	return DomainXds
}

type EnvoyConfig struct {
	ClusterName string `json:"cluster_name"`
}

type XdsMarker struct {
	Type ControllerType `json:"type"`
}

func (x *XdsMarker) GetDomain() string {
	return DomainXds
}
