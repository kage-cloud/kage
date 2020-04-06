module github.com/kage-cloud/kage

go 1.13

require (
	github.com/kage-cloud/kage/xds v0.0.0-20200405002512-692a1bd05fc3 // indirect
	github.com/stretchr/testify v1.5.1
	k8s.io/api v0.15.10
	k8s.io/apimachinery v0.15.10
	k8s.io/client-go v0.15.10
)

replace k8s.io/api => k8s.io/api v0.15.10
