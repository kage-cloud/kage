package meta

type ControllerType string

const (
	SourceControllerType ControllerType = "source"
	CanaryControllerType ControllerType = "canary"
)

type MeshMarkerLabel struct {
	IsMesh bool `json:"is_mesh"`
}

func (m *MeshMarkerLabel) GetDomain() string {
	return DomainXds
}

type Xds struct {
	Name          string            `json:"name"`
	LabelSelector map[string]string `json:"label_selector"`
	Config        XdsConfig         `json:"config"`
}

func (x *Xds) GetDomain() string {
	return DomainXds
}

type XdsConfig struct {
	NodeId string      `json:"node_id"`
	Canary EnvoyConfig `json:"canary"`
	Source EnvoyConfig `json:"source"`
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
