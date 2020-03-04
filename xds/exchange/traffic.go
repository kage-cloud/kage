package exchange

type DirectTrafficRequest struct {
	EndpointName string `json:"endpoint_name" param:"endpoint_name"`
	Percentage   uint32 `json:"percentage"`
}

type DirectTrafficResponse struct {
}
