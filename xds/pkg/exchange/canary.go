package exchange

type Canary struct {
	Name              string `json:"name"`
	TargetDeploy      string `json:"target_deploy"`
	RoutingPercentage uint32 `json:"routing_percentage"`
}

type CreateCanaryRequest struct {
	Name                    string `param:"name"`
	Namespace               string `param:"namespace"`
	CanaryRoutingPercentage uint32 `json:"canary_routing_percentage"`
}

type CreateCanaryResponse struct {
	Data *Canary `json:"data"`
}

type DeleteCanaryRequest struct {
	Name      string `param:"name"`
	Namespace string `param:"namespace"`
}
