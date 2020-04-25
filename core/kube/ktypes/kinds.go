package ktypes

type Kind string

const (
	KindPod        Kind = "Pod"
	KindDeploy     Kind = "Deploy"
	KindService    Kind = "Service"
	KindReplicaSet Kind = "ReplicaSet"
	KindConfigMap  Kind = "ConfigMap"
	KindEndpoints  Kind = "Endpoints"
)
