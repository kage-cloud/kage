package exchange

import "github.com/kage-cloud/kage/core/except"

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

func (c *CreateCanaryRequest) Validate() error {
	if c.Name == "" {
		return except.NewError("Name field is required.", except.ErrInvalid)
	}
	if c.Namespace == "" {
		return except.NewError("Namespace field is required.", except.ErrInvalid)
	}
	return nil
}

type CreateCanaryResponse struct {
	Data *Canary `json:"data"`
}

type DeleteCanaryRequest struct {
	Name      string `param:"name"`
	Namespace string `param:"namespace"`
}
