package exchange

type GetAdminRequest struct {
	Namespace  string `param:"namespace"`
	CanaryName string `param:"canary_name"`
}
