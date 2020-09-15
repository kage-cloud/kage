module github.com/kage-cloud/kage/xds

go 1.14

require (
	github.com/eddieowens/axon v0.6.0
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/kage-cloud/kage/core v0.0.0-00010101000000-000000000000
	github.com/labstack/echo/v4 v4.1.15
	github.com/rancher/k3d/v3 v3.0.1
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.4.0
	google.golang.org/grpc v1.29.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	k8s.io/utils v0.0.0-20200109141947-94aeca20bf09
)

replace github.com/kage-cloud/kage/core => ../core
