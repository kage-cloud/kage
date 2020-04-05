package xds

type RoutingSpec struct {
	CanaryName          string
	TargetName          string
	CanaryRoutingWeight uint32
	TotalRoutingWeight  uint32
}
