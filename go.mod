module github.com/kage-cloud/kage

go 1.14

replace k8s.io/api => k8s.io/api v0.15.10

replace k8s.io/apimachinery => k8s.io/apimachinery v0.15.10

replace k8s.io/client-go => k8s.io/client-go v0.15.10

require (
	github.com/bxcodec/faker/v3 v3.5.0 // indirect
	github.com/fatih/structtag v1.2.0
	github.com/stretchr/testify v1.6.1
)
