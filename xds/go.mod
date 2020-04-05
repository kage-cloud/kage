module github.com/kage-cloud/kage/xds

go 1.13

require (
	github.com/eddieowens/axon v0.6.0
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/kage-cloud/kage v0.0.0-20200326012602-b8376b4144b5
	github.com/labstack/echo/v4 v4.1.15
	github.com/stretchr/testify v1.5.1 // indirect
	google.golang.org/grpc v1.27.1
	k8s.io/api v0.15.10
	k8s.io/apimachinery v0.15.10
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
)

replace github.com/kage-cloud/kage => ../
