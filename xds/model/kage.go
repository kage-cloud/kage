package model

type KageSpec struct {
	TargetDeployName        string
	CanaryRoutingPercentage uint32
}

type DeleteKageSpec struct {
	TargetDeployName string
}
