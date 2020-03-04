module github.com/eddieowens/kage/xds

go 1.13

require (
	github.com/eddieowens/axon v0.6.0
	github.com/eddieowens/kage v0.0.0-00010101000000-000000000000
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/labstack/echo/v4 v4.1.15
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.15.10
	k8s.io/apimachinery v0.15.10
)

replace github.com/eddieowens/kage => ../
