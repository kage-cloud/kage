package model

type Baseline struct {
	NodeId         string
	NodeCluster    string
	XdsAddress     string
	XdsPort        uint16
	XdsClusterName string
	AdminPort      uint16

	ServiceClusterName string
	CanaryClusterName  string
}
